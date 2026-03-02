package queues

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/modules/sql"
	"github.com/awesome-goose/goose/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Queue is the job queue service
type Queue struct {
	db     *sql.Db   `inject:""`
	log    types.Log `inject:""`
	config *Config   `inject:""`

	// Worker management
	mu             sync.Mutex
	activeWorkers  map[string]int                  // key: "queue:job"
	workers        map[string][]context.CancelFunc // key: "queue:job"
	isShuttingDown bool
	wg             sync.WaitGroup

	// Metrics
	metrics *QueueMetrics
}

// QueueMetrics tracks queue statistics
type QueueMetrics struct {
	mu              sync.RWMutex
	TotalProcessed  int64
	TotalSucceeded  int64
	TotalFailed     int64
	TotalRetried    int64
	ProcessingTimes []time.Duration
}

// GetMetrics returns a copy of the current metrics
func (q *Queue) GetMetrics() QueueMetrics {
	if q.metrics == nil {
		return QueueMetrics{}
	}
	q.metrics.mu.RLock()
	defer q.metrics.mu.RUnlock()
	return QueueMetrics{
		TotalProcessed:  q.metrics.TotalProcessed,
		TotalSucceeded:  q.metrics.TotalSucceeded,
		TotalFailed:     q.metrics.TotalFailed,
		TotalRetried:    q.metrics.TotalRetried,
		ProcessingTimes: append([]time.Duration{}, q.metrics.ProcessingTimes...),
	}
}

// workerKey returns the key for a queue/job combination
func workerKey(queue, job string) string {
	return fmt.Sprintf("%s:%s", queue, job)
}

// queueName returns the configured queue or default
func (q *Queue) queueName() string {
	if q.config != nil && q.config.Queue != "" {
		return q.config.Queue
	}
	return DefaultQueue
}

// Push adds a new job to the queue.
// If the queue doesn't exist, it will be created.
//
// Parameters:
//   - queueName: The name of the queue
//   - jobName: The name/type of the job
//   - data: The job payload data
//   - config: Optional job configuration
//
// Returns the created job.
func (q *Queue) Push(queueName string, jobName string, data any, config *JobConfig) (*QueueJob, error) {
	if queueName == "" {
		queueName = q.queueName()
	}

	// Marshal data to JSON
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Marshal config to JSON if provided
	var configBytes json.RawMessage
	if config != nil {
		configBytes, err = json.Marshal(config)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()

	// Upsert the queue (create if not exists)
	queue := &QueueQueue{
		Id:        uuid.New().String(),
		Name:      queueName,
		Priority:  DefaultPriority,
		Status:    QueueStatusActive,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	err = q.db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(queue).Error
	if err != nil {
		return nil, err
	}

	// Get the existing queue
	var existingQueue QueueQueue
	err = q.db.DB.Where("name = ?", queueName).First(&existingQueue).Error
	if err != nil {
		return nil, err
	}

	// Set job config defaults
	priority := DefaultPriority
	retryLimit := DefaultRetryLimit
	retryDelay := DefaultRetryDelay
	var startAt, expireAt *time.Time

	if config != nil {
		if config.Priority != 0 {
			priority = config.Priority
		}
		if config.RetryLimit != 0 {
			retryLimit = config.RetryLimit
		}
		if config.RetryDelay != 0 {
			retryDelay = config.RetryDelay
		}
		startAt = config.StartAt
		expireAt = config.ExpireAt

		// Handle singleton jobs - don't create if one already exists
		if config.Singleton {
			var existingJob QueueJob
			err = q.db.DB.Where("queue_id = ? AND name = ? AND status IN ?",
				existingQueue.Id, jobName,
				[]string{JobStatusNew, JobStatusInProgress, JobStatusRetrying},
			).First(&existingJob).Error
			if err == nil {
				// Job already exists, return it
				return &existingJob, nil
			}
			if !stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
		}
	}

	// Apply module config defaults if not specified
	if q.config != nil {
		if retryLimit == DefaultRetryLimit && q.config.DefaultRetryLimit > 0 {
			retryLimit = q.config.DefaultRetryLimit
		}
		if retryDelay == DefaultRetryDelay && q.config.DefaultRetryDelay > 0 {
			retryDelay = q.config.DefaultRetryDelay
		}
	}

	// Create the job
	job := &QueueJob{
		Id:         uuid.New().String(),
		QueueId:    existingQueue.Id,
		Name:       jobName,
		Priority:   priority,
		Data:       dataBytes,
		Config:     configBytes,
		RetryLimit: retryLimit,
		RetryDelay: retryDelay,
		RetryCount: 0,
		StartAt:    startAt,
		ExpireAt:   expireAt,
		Status:     JobStatusNew,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}

	err = q.db.DB.Create(job).Error
	if err != nil {
		return nil, err
	}

	return job, nil
}

// Delay pushes a job to be executed after a delay
// This is a convenience method equivalent to Push with StartAt set
func (q *Queue) Delay(queueName, jobName string, data any, delay time.Duration, config *JobConfig) (*QueueJob, error) {
	if config == nil {
		config = &JobConfig{}
	}
	startAt := time.Now().UTC().Add(delay)
	config.StartAt = &startAt
	return q.Push(queueName, jobName, data, config)
}

// At pushes a job to be executed at a specific time
func (q *Queue) At(queueName, jobName string, data any, at time.Time, config *JobConfig) (*QueueJob, error) {
	if config == nil {
		config = &JobConfig{}
	}
	config.StartAt = &at
	return q.Push(queueName, jobName, data, config)
}

// Later pushes a job with common delay presets
func (q *Queue) Later(queueName, jobName string, data any, config *JobConfig) *DelayedPush {
	return &DelayedPush{
		q:         q,
		queueName: queueName,
		jobName:   jobName,
		data:      data,
		config:    config,
	}
}

// DelayedPush provides fluent API for delayed job pushing
type DelayedPush struct {
	q         *Queue
	queueName string
	jobName   string
	data      any
	config    *JobConfig
}

// InSeconds pushes the job after N seconds
func (d *DelayedPush) InSeconds(n int) (*QueueJob, error) {
	return d.q.Delay(d.queueName, d.jobName, d.data, time.Duration(n)*time.Second, d.config)
}

// InMinutes pushes the job after N minutes
func (d *DelayedPush) InMinutes(n int) (*QueueJob, error) {
	return d.q.Delay(d.queueName, d.jobName, d.data, time.Duration(n)*time.Minute, d.config)
}

// InHours pushes the job after N hours
func (d *DelayedPush) InHours(n int) (*QueueJob, error) {
	return d.q.Delay(d.queueName, d.jobName, d.data, time.Duration(n)*time.Hour, d.config)
}

// Tomorrow pushes the job to run tomorrow at the same time
func (d *DelayedPush) Tomorrow() (*QueueJob, error) {
	return d.q.Delay(d.queueName, d.jobName, d.data, 24*time.Hour, d.config)
}

// PushBatch pushes multiple jobs at once in a single transaction
// All jobs go to the same queue but can have different names and data
type BatchJob struct {
	JobName string
	Data    any
	Config  *JobConfig
}

// PushBatch pushes multiple jobs in a single transaction
func (q *Queue) PushBatch(queueName string, jobs []BatchJob) ([]*QueueJob, error) {
	if len(jobs) == 0 {
		return nil, nil
	}

	if queueName == "" {
		queueName = q.queueName()
	}

	now := time.Now().UTC()

	// Ensure queue exists
	queue := &QueueQueue{
		Id:        uuid.New().String(),
		Name:      queueName,
		Priority:  DefaultPriority,
		Status:    QueueStatusActive,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	err := q.db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(queue).Error
	if err != nil {
		return nil, err
	}

	var existingQueue QueueQueue
	err = q.db.DB.Where("name = ?", queueName).First(&existingQueue).Error
	if err != nil {
		return nil, err
	}

	// Create all jobs in a transaction
	var createdJobs []*QueueJob

	err = q.db.DB.Transaction(func(tx *gorm.DB) error {
		for _, bj := range jobs {
			dataBytes, err := json.Marshal(bj.Data)
			if err != nil {
				return err
			}

			var configBytes json.RawMessage
			if bj.Config != nil {
				configBytes, err = json.Marshal(bj.Config)
				if err != nil {
					return err
				}
			}

			priority := DefaultPriority
			retryLimit := DefaultRetryLimit
			retryDelay := DefaultRetryDelay
			var startAt, expireAt *time.Time

			if bj.Config != nil {
				if bj.Config.Priority != 0 {
					priority = bj.Config.Priority
				}
				if bj.Config.RetryLimit != 0 {
					retryLimit = bj.Config.RetryLimit
				}
				if bj.Config.RetryDelay != 0 {
					retryDelay = bj.Config.RetryDelay
				}
				startAt = bj.Config.StartAt
				expireAt = bj.Config.ExpireAt
			}

			job := &QueueJob{
				Id:         uuid.New().String(),
				QueueId:    existingQueue.Id,
				Name:       bj.JobName,
				Priority:   priority,
				Data:       dataBytes,
				Config:     configBytes,
				RetryLimit: retryLimit,
				RetryDelay: retryDelay,
				RetryCount: 0,
				StartAt:    startAt,
				ExpireAt:   expireAt,
				Status:     JobStatusNew,
				CreatedAt:  &now,
				UpdatedAt:  &now,
			}

			if err := tx.Create(job).Error; err != nil {
				return err
			}
			createdJobs = append(createdJobs, job)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdJobs, nil
}

// Pop retrieves and locks the next available job from the queue.
// Uses FOR UPDATE SKIP LOCKED for concurrent-safe job fetching.
//
// Parameters:
//   - queueName: The name of the queue
//   - jobName: The name/type of the job to pop
//
// Returns the job if found, nil otherwise.
func (q *Queue) Pop(queueName string, jobName string) (*QueueJob, error) {
	if queueName == "" {
		queueName = q.queueName()
	}

	// Get the queue
	var existingQueue QueueQueue
	err := q.db.DB.Where("name = ?", queueName).First(&existingQueue).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No queue found, return nil
		}
		return nil, err
	}

	var poppedJob *QueueJob
	now := time.Now().UTC()

	// Use transaction with row-level locking
	err = q.db.DB.Transaction(func(tx *gorm.DB) error {
		var job QueueJob

		// Query for available job with row locking
		// Uses FOR UPDATE SKIP LOCKED to avoid blocking on locked rows
		result := tx.Raw(`
			SELECT j.* FROM queue_jobs j
			JOIN queue_queues q ON q.id = j.queue_id
			WHERE q.name = ?
				AND j.name = ?
				AND j.status = ?
				AND (j.start_at IS NULL OR j.start_at <= NOW())
				AND (j.expire_at IS NULL OR j.expire_at > NOW())
			ORDER BY j.priority DESC, j.created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		`, queueName, jobName, JobStatusNew).Scan(&job)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return nil // No job available
		}

		// Update job status to in_progress
		err := tx.Model(&QueueJob{}).
			Where("id = ?", job.Id).
			Updates(map[string]any{
				"status":     JobStatusInProgress,
				"started_at": &now,
				"updated_at": &now,
			}).Error
		if err != nil {
			return err
		}

		job.Status = JobStatusInProgress
		job.StartedAt = &now
		job.UpdatedAt = &now
		poppedJob = &job

		return nil
	})

	if err != nil {
		return nil, err
	}

	return poppedJob, nil
}

// Log records the result of a job execution.
// Updates the job status and creates a log entry.
//
// Parameters:
//   - jobId: The ID of the job
//   - status: The result status (success, failed)
//   - output: The output/result data
//
// Returns the updated job.
func (q *Queue) Log(jobId string, status string, output any) (*QueueJob, error) {
	// Validate status
	if status != JobStatusSuccess && status != JobStatusFailed {
		return nil, errors.ErrQueueInvalidStatus.WithDetail(status)
	}

	// Get the job
	var job QueueJob
	err := q.db.DB.Where("id = ?", jobId).First(&job).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrQueueJobNotFound
		}
		return nil, err
	}

	now := time.Now().UTC()

	// Marshal output to JSON
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}

	// Update job status
	err = q.db.DB.Model(&QueueJob{}).
		Where("id = ?", jobId).
		Updates(map[string]any{
			"status":     status,
			"expired_at": &now,
			"updated_at": &now,
		}).Error
	if err != nil {
		return nil, err
	}

	// Create log entry
	logEntry := &QueueLog{
		Id:        uuid.New().String(),
		QueueId:   job.QueueId,
		JobId:     job.Id,
		Output:    outputBytes,
		Status:    status,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	err = q.db.DB.Create(logEntry).Error
	if err != nil {
		return nil, err
	}

	job.Status = status
	job.ExpiredAt = &now
	job.UpdatedAt = &now

	return &job, nil
}

// Retry marks a job for retry.
// Increments the retry count and resets the status to new.
//
// Parameters:
//   - jobId: The ID of the job to retry
//
// Returns error if job not found or max retries exceeded.
func (q *Queue) Retry(jobId string) error {
	var job QueueJob
	err := q.db.DB.Where("id = ?", jobId).First(&job).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.ErrQueueJobNotFound
		}
		return err
	}

	// Check if retry limit exceeded
	if job.RetryCount >= job.RetryLimit {
		return errors.ErrQueueRetryExhausted
	}

	now := time.Now().UTC()
	retryAt := now.Add(time.Duration(job.RetryDelay) * time.Millisecond)

	// Update job for retry
	return q.db.DB.Model(&QueueJob{}).
		Where("id = ?", jobId).
		Updates(map[string]any{
			"status":      JobStatusNew,
			"retry_count": job.RetryCount + 1,
			"start_at":    &retryAt,
			"started_at":  nil,
			"updated_at":  &now,
		}).Error
}

// GetJobData unmarshals the job data into the provided type
func GetJobData[T any](job *QueueJob) (T, error) {
	var data T
	if job.Data == nil {
		return data, nil
	}
	err := json.Unmarshal(job.Data, &data)
	return data, err
}

// GetQueue retrieves a queue by name
func (q *Queue) GetQueue(name string) (*QueueQueue, error) {
	var queue QueueQueue
	err := q.db.DB.Where("name = ?", name).First(&queue).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrQueueNotFound
		}
		return nil, err
	}
	return &queue, nil
}

// GetJob retrieves a job by ID
func (q *Queue) GetJob(id string) (*QueueJob, error) {
	var job QueueJob
	err := q.db.DB.Where("id = ?", id).First(&job).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrQueueJobNotFound
		}
		return nil, err
	}
	return &job, nil
}

// GetJobLogs retrieves all logs for a job
func (q *Queue) GetJobLogs(jobId string) ([]QueueLog, error) {
	var logs []QueueLog
	err := q.db.DB.Where("job_id = ?", jobId).Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// ListJobs retrieves jobs with optional filtering
func (q *Queue) ListJobs(queueName string, status string, limit int) ([]QueueJob, error) {
	query := q.db.DB.Model(&QueueJob{})

	if queueName != "" {
		var queue QueueQueue
		err := q.db.DB.Where("name = ?", queueName).First(&queue).Error
		if err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.ErrQueueNotFound
			}
			return nil, err
		}
		query = query.Where("queue_id = ?", queue.Id)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if limit <= 0 {
		limit = 100
	}

	var jobs []QueueJob
	err := query.Order("priority DESC, created_at ASC").Limit(limit).Find(&jobs).Error
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

// Cancel cancels a pending job
func (q *Queue) Cancel(jobId string) error {
	var job QueueJob
	err := q.db.DB.Where("id = ?", jobId).First(&job).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.ErrQueueJobNotFound
		}
		return err
	}

	// Can only cancel new jobs
	if job.Status != JobStatusNew {
		return errors.ErrQueueJobCancelNotAllowed
	}

	now := time.Now().UTC()
	return q.db.DB.Model(&QueueJob{}).
		Where("id = ?", jobId).
		Updates(map[string]any{
			"status":     JobStatusExpired,
			"expired_at": &now,
			"updated_at": &now,
		}).Error
}

// PauseQueue pauses a queue (stops new jobs from being popped)
func (q *Queue) PauseQueue(name string) error {
	return q.db.DB.Model(&QueueQueue{}).
		Where("name = ?", name).
		Update("status", QueueStatusPaused).Error
}

// ResumeQueue resumes a paused queue
func (q *Queue) ResumeQueue(name string) error {
	return q.db.DB.Model(&QueueQueue{}).
		Where("name = ?", name).
		Update("status", QueueStatusActive).Error
}

// Initialize initializes the worker tracking maps
func (q *Queue) Initialize() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.activeWorkers == nil {
		q.activeWorkers = make(map[string]int)
	}
	if q.workers == nil {
		q.workers = make(map[string][]context.CancelFunc)
	}
}

// Process starts processing jobs from a queue with auto-scaling workers.
// It spawns min workers initially and scales up to max based on demand.
// Workers self-terminate when idle for idleThreshold consecutive polls.
//
// Parameters:
//   - handler: The job handler configuration
//
// This method is non-blocking and returns immediately after spawning workers.
func (q *Queue) Process(handler *JobHandler) {
	q.Initialize()

	// Initialize metrics if not already done
	q.mu.Lock()
	if q.metrics == nil {
		q.metrics = &QueueMetrics{}
	}
	q.mu.Unlock()

	queueName := handler.Queue
	jobName := handler.Job
	fn := handler.Handler
	min := handler.MinWorkers
	max := handler.MaxWorkers
	pollMs := handler.PollIntervalMs
	idleThreshold := handler.IdleThreshold
	timeoutMs := handler.TimeoutMs
	useExpBackoff := handler.UseExponentialBackoff

	// Apply defaults
	if min <= 0 {
		min = DefaultMinWorkers
	}
	if max <= 0 {
		max = DefaultMaxWorkers
	}
	if pollMs <= 0 {
		pollMs = DefaultPollInterval
	}
	if idleThreshold <= 0 {
		idleThreshold = DefaultIdleThreshold
	}
	if timeoutMs <= 0 {
		timeoutMs = DefaultJobTimeout
	}

	key := workerKey(queueName, jobName)

	// Spawn initial workers
	for i := 0; i < min; i++ {
		q.spawnWorker(key, queueName, jobName, fn, min, pollMs, idleThreshold, timeoutMs, useExpBackoff)
	}

	// Start auto-scaler goroutine
	go q.autoScaler(key, queueName, jobName, fn, min, max, pollMs, idleThreshold, timeoutMs, useExpBackoff)
}

// spawnWorker creates a new worker goroutine
func (q *Queue) spawnWorker(key, queueName, jobName string, fn JobHandlerFn, min, pollMs, idleThreshold, timeoutMs int, useExpBackoff bool) {
	q.mu.Lock()
	if q.isShuttingDown {
		q.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	q.workers[key] = append(q.workers[key], cancel)
	q.activeWorkers[key]++
	q.wg.Add(1)
	q.mu.Unlock()

	go func() {
		defer func() {
			// Panic recovery
			if r := recover(); r != nil {
				if q.log != nil {
					q.log.Error("worker panic recovered", "queue", queueName, "job", jobName, "panic", r, "stack", string(debug.Stack()))
				}
			}
			q.wg.Done()
			q.mu.Lock()
			q.activeWorkers[key]--
			q.mu.Unlock()
		}()

		idleCount := 0
		sleepDuration := time.Duration(pollMs) * time.Millisecond

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if q.isShuttingDown {
				return
			}

			// Pop the next job
			job, err := q.Pop(queueName, jobName)
			if err != nil {
				if q.log != nil {
					q.log.Error("failed to pop job", "queue", queueName, "job", jobName, "error", err)
				}
				time.Sleep(sleepDuration)
				continue
			}

			if job == nil {
				idleCount++

				// Check if we should self-terminate
				q.mu.Lock()
				currentWorkers := q.activeWorkers[key]
				q.mu.Unlock()

				if currentWorkers > min && idleCount >= idleThreshold {
					return
				}

				time.Sleep(sleepDuration)
				continue
			}

			// Reset idle count when we get a job
			idleCount = 0

			// Process the job with retries and timeout
			q.processJob(ctx, job, fn, timeoutMs, useExpBackoff)
		}
	}()
}

// processJob handles a single job with retries, timeout, and panic recovery
func (q *Queue) processJob(parentCtx context.Context, job *QueueJob, fn JobHandlerFn, timeoutMs int, useExpBackoff bool) {
	startTime := time.Now()

	retryLimit := job.RetryLimit
	if retryLimit <= 0 {
		retryLimit = DefaultRetryLimit
	}
	retryDelay := job.RetryDelay
	if retryDelay <= 0 {
		retryDelay = DefaultRetryDelay
	}

	var success bool
	var lastErr error
	var lastResult any

	for attempt := 0; attempt <= retryLimit && !success; attempt++ {
		if attempt > 0 {
			// Calculate delay with optional exponential backoff
			delay := retryDelay
			if useExpBackoff {
				delay = retryDelay * (1 << (attempt - 1)) // 2^(attempt-1) * base delay
				if delay > DefaultMaxBackoffDelay {
					delay = DefaultMaxBackoffDelay
				}
			}
			time.Sleep(time.Duration(delay) * time.Millisecond)

			// Update metrics
			if q.metrics != nil {
				q.metrics.mu.Lock()
				q.metrics.TotalRetried++
				q.metrics.mu.Unlock()
			}
		}

		// Execute with timeout and panic recovery
		result, err := q.executeJobWithTimeout(parentCtx, job, fn, timeoutMs)

		if err == nil {
			lastResult = result
			success = true
		} else {
			lastErr = err
			if q.log != nil {
				q.log.Warning("job attempt failed",
					"job_id", job.Id,
					"queue", job.Name,
					"attempt", attempt+1,
					"max_attempts", retryLimit+1,
					"error", err)
			}
		}
	}

	elapsed := time.Since(startTime)

	// Update metrics
	if q.metrics != nil {
		q.metrics.mu.Lock()
		q.metrics.TotalProcessed++
		if success {
			q.metrics.TotalSucceeded++
		} else {
			q.metrics.TotalFailed++
		}
		// Keep last 100 processing times
		if len(q.metrics.ProcessingTimes) >= 100 {
			q.metrics.ProcessingTimes = q.metrics.ProcessingTimes[1:]
		}
		q.metrics.ProcessingTimes = append(q.metrics.ProcessingTimes, elapsed)
		q.metrics.mu.Unlock()
	}

	// Log final result
	if success {
		q.Log(job.Id, JobStatusSuccess, map[string]any{
			"result":  lastResult,
			"elapsed": elapsed.String(),
		})
	} else {
		errMsg := "unknown error"
		if lastErr != nil {
			errMsg = lastErr.Error()
		}
		q.Log(job.Id, JobStatusFailed, map[string]any{
			"error":   errMsg,
			"elapsed": elapsed.String(),
		})
	}
}

// executeJobWithTimeout runs a job handler with timeout and panic recovery
func (q *Queue) executeJobWithTimeout(parentCtx context.Context, job *QueueJob, fn JobHandlerFn, timeoutMs int) (result any, err error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Channel for result
	done := make(chan struct{})

	go func() {
		defer func() {
			// Recover from panic in handler
			if r := recover(); r != nil {
				err = fmt.Errorf("job handler panicked: %v\n%s", r, debug.Stack())
				if q.log != nil {
					q.log.Error("job handler panic", "job_id", job.Id, "panic", r, "stack", string(debug.Stack()))
				}
			}
			close(done)
		}()

		result, err = fn(ctx, job)
	}()

	select {
	case <-done:
		return result, err
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return nil, errors.ErrQueueJobTimeout
		}
		return nil, ctx.Err()
	}
}

// autoScaler monitors and scales workers based on demand
func (q *Queue) autoScaler(key, queueName, jobName string, fn JobHandlerFn, min, max, pollMs, idleThreshold, timeoutMs int, useExpBackoff bool) {
	ticker := time.NewTicker(time.Duration(pollMs) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if q.isShuttingDown {
			return
		}

		q.mu.Lock()
		currentWorkers := q.activeWorkers[key]
		q.mu.Unlock()

		// Scale up if we have room
		if currentWorkers < max {
			q.spawnWorker(key, queueName, jobName, fn, min, pollMs, idleThreshold, timeoutMs, useExpBackoff)
		}
	}
}

// Shutdown gracefully shuts down all workers
// Waits for all workers to finish processing their current jobs
func (q *Queue) Shutdown() {
	q.mu.Lock()
	q.isShuttingDown = true

	// Cancel all worker contexts
	for _, cancels := range q.workers {
		for _, cancel := range cancels {
			cancel()
		}
	}
	q.mu.Unlock()

	// Wait for all workers to finish
	q.wg.Wait()
}

// ActiveWorkerCount returns the number of active workers for a queue/job
func (q *Queue) ActiveWorkerCount(queueName, jobName string) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.activeWorkers[workerKey(queueName, jobName)]
}

// TotalActiveWorkers returns the total number of active workers across all queues
func (q *Queue) TotalActiveWorkers() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	total := 0
	for _, count := range q.activeWorkers {
		total += count
	}
	return total
}
