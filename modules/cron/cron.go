package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/modules/sql"
	"github.com/awesome-goose/goose/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Cron is the cron job service
type Cron struct {
	db     *sql.Db   `inject:""`
	log    types.Log `inject:""`
	config *Config   `inject:""`

	// Runner management
	mu             sync.Mutex
	isRunning      bool
	isShuttingDown bool
	stopChan       chan struct{}
	wg             sync.WaitGroup

	// Handlers map: "group:name" -> handler
	handlers map[string]*CronHandler

	// Metrics
	metrics *CronMetrics
}

// CronMetrics tracks cron statistics
type CronMetrics struct {
	mu             sync.RWMutex
	TotalRuns      int64
	TotalSucceeded int64
	TotalFailed    int64
	TotalRetried   int64
	LastRunAt      *time.Time
}

// GetMetrics returns a copy of the current metrics
func (c *Cron) GetMetrics() CronMetrics {
	if c.metrics == nil {
		return CronMetrics{}
	}
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()
	return CronMetrics{
		TotalRuns:      c.metrics.TotalRuns,
		TotalSucceeded: c.metrics.TotalSucceeded,
		TotalFailed:    c.metrics.TotalFailed,
		TotalRetried:   c.metrics.TotalRetried,
		LastRunAt:      c.metrics.LastRunAt,
	}
}

// handlerKey returns the key for a group/name combination
func handlerKey(group, name string) string {
	return fmt.Sprintf("%s:%s", group, name)
}

// Register registers a cron job in the database.
// If the job already exists (same group+name), it will not be modified.
//
// Parameters:
//   - group: The group/category of the job
//   - name: The unique name of the job within the group
//   - pattern: The cron expression
//   - config: Optional job configuration
//
// Returns the registered or existing job.
func (c *Cron) Register(group string, name string, pattern string, config *CronConfig) (*CronJob, error) {
	// Validate cron pattern
	if !IsValidCronPattern(pattern) {
		return nil, errors.ErrCronInvalidPattern
	}

	now := time.Now().UTC()

	// Build job with defaults
	job := &CronJob{
		Group:      group,
		Name:       name,
		Pattern:    pattern,
		Priority:   DefaultPriority,
		RetryLimit: DefaultRetryLimit,
		RetryDelay: DefaultRetryDelay,
		Status:     JobStatusPending,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}

	// Apply config if provided
	if config != nil {
		if config.Priority != 0 {
			job.Priority = config.Priority
		}
		if config.RetryLimit != 0 {
			job.RetryLimit = config.RetryLimit
		}
		if config.RetryDelay != 0 {
			// Cap retry delay at max allowed
			if config.RetryDelay > DefaultMaxRetryDelay {
				job.RetryDelay = DefaultMaxRetryDelay
			} else {
				job.RetryDelay = config.RetryDelay
			}
		}
		job.StartAt = config.StartAt
		job.ExpireAt = config.ExpireAt

		// Store config as JSON
		configBytes, err := json.Marshal(config)
		if err != nil {
			return nil, err
		}
		job.Config = configBytes
	}

	// Upsert: Insert if not exists, do nothing on conflict
	err := c.db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "group"}, {Name: "name"}},
		DoNothing: true,
	}).Create(job).Error
	if err != nil {
		return nil, err
	}

	// Return the existing job
	var existingJob CronJob
	err = c.db.DB.Where(`"group" = ? AND name = ?`, group, name).First(&existingJob).Error
	if err != nil {
		return nil, err
	}

	return &existingJob, nil
}

// Select fetches a pending job for processing with row locking.
// Uses FOR UPDATE SKIP LOCKED to allow concurrent workers.
//
// Parameters:
//   - group: The group of the job
//   - name: The name of the job
//
// Returns the job if available and ready to run, nil otherwise.
func (c *Cron) Select(group string, name string) (*CronJob, error) {
	var job *CronJob

	err := c.db.DB.Transaction(func(tx *gorm.DB) error {
		// Use raw query for FOR UPDATE SKIP LOCKED
		result := tx.Raw(`
			SELECT * FROM cron_jobs j
			WHERE j."group" = ?
			  AND j.name = ?
			  AND j.status = ?
			  AND j.deleted_at IS NULL
			  AND (j.start_at IS NULL OR j.start_at <= NOW())
			  AND (j.expire_at IS NULL OR j.expire_at > NOW())
			ORDER BY j.priority DESC, j.created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		`, group, name, JobStatusPending).Scan(&job)

		if result.Error != nil {
			return result.Error
		}

		if job == nil {
			return nil // No job available
		}

		// Update job status to in_progress
		now := time.Now().UTC()
		err := tx.Model(&CronJob{}).
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

		return nil
	})

	return job, err
}

// Log records the result of a cron job execution and resets the job to pending.
//
// Parameters:
//   - jobId: The ID of the job
//   - status: "success" or "failed"
//   - output: The output/result data
//
// Returns the created log entry.
func (c *Cron) Log(jobId string, status string, output any) (*CronLog, error) {
	// Validate status
	if status != LogStatusSuccess && status != LogStatusFailed {
		return nil, fmt.Errorf("invalid log status: %s", status)
	}

	// Find the job
	var job CronJob
	err := c.db.DB.Where("id = ?", jobId).First(&job).Error
	if err != nil {
		return nil, errors.ErrCronJobNotFound
	}

	// Marshal output
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}

	// Create log entry
	log := &CronLog{
		JobId:  job.Id,
		Output: outputBytes,
		Status: status,
	}

	err = c.db.DB.Create(log).Error
	if err != nil {
		return nil, err
	}

	// Reset job status to pending for next cron run
	now := time.Now().UTC()
	err = c.db.DB.Model(&CronJob{}).
		Where("id = ?", job.Id).
		Updates(map[string]any{
			"status":     JobStatusPending,
			"started_at": nil,
			"updated_at": &now,
		}).Error
	if err != nil {
		return nil, err
	}

	return log, nil
}

// Start begins the cron runner that checks and executes jobs at regular intervals.
// This method blocks until Stop is called or the context is cancelled.
func (c *Cron) Start(ctx context.Context, handlers []*CronHandler) error {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return nil // Already running
	}
	c.isRunning = true
	c.isShuttingDown = false
	c.stopChan = make(chan struct{})
	c.handlers = make(map[string]*CronHandler)
	c.metrics = &CronMetrics{}
	c.mu.Unlock()

	// Register all handlers
	for _, h := range handlers {
		_, err := c.Register(h.Group, h.Name, h.Pattern, h.Config)
		if err != nil {
			c.log.Warning(fmt.Sprintf("Failed to register cron job %s/%s: %v", h.Group, h.Name, err))
			continue
		}
		c.handlers[handlerKey(h.Group, h.Name)] = h
	}

	// Get tick interval
	tickInterval := c.config.TickInterval
	if tickInterval == 0 {
		tickInterval = time.Duration(DefaultTickInterval) * time.Second
	}

	// Get timezone
	tz := c.config.Timezone
	if tz == "" {
		tz = "Etc/UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	c.log.Info(fmt.Sprintf("Cron runner started with %d handlers, tick interval: %v", len(handlers), tickInterval))

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Cron runner stopped (context cancelled)")
			return ctx.Err()
		case <-c.stopChan:
			c.log.Info("Cron runner stopped")
			return nil
		case <-ticker.C:
			c.runTick(loc)
		}
	}
}

// Stop stops the cron runner gracefully
func (c *Cron) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return
	}

	c.isShuttingDown = true
	close(c.stopChan)

	// Wait for any running jobs to complete
	c.wg.Wait()
	c.isRunning = false
}

// runTick runs one tick of the cron runner
func (c *Cron) runTick(loc *time.Location) {
	now := time.Now().In(loc)

	c.mu.Lock()
	if c.metrics != nil {
		c.metrics.mu.Lock()
		c.metrics.LastRunAt = &now
		c.metrics.mu.Unlock()
	}
	handlers := c.handlers
	c.mu.Unlock()

	for _, h := range handlers {
		// Check if should run now based on pattern
		if !ShouldRunNow(h.Pattern, now) {
			continue
		}

		c.wg.Add(1)
		go c.processJob(h)
	}
}

// processJob processes a single cron job with retry logic
func (c *Cron) processJob(handler *CronHandler) {
	defer c.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			c.log.Error(fmt.Sprintf("Panic in cron job %s/%s: %v\n%s", handler.Group, handler.Name, r, debug.Stack()))
			c.updateMetrics(false, true)
		}
	}()

	// Try to select the job
	job, err := c.Select(handler.Group, handler.Name)
	if err != nil {
		c.log.Warning(fmt.Sprintf("Failed to select cron job %s/%s: %v", handler.Group, handler.Name, err))
		return
	}
	if job == nil {
		// Job not available (already running or doesn't exist)
		return
	}

	// Get retry settings
	retryLimit := job.RetryLimit
	if retryLimit == 0 {
		retryLimit = DefaultRetryLimit
	}
	retryDelay := job.RetryDelay
	if retryDelay == 0 || retryDelay > DefaultMaxRetryDelay {
		retryDelay = DefaultRetryDelay
	}

	// Execute with retries
	var success bool
	var lastError error
	var result any

	for attempt := 1; attempt <= retryLimit+1; attempt++ {
		c.updateMetrics(false, false)

		result, lastError = c.executeJobWithTimeout(handler, job)
		if lastError == nil {
			success = true
			break
		}

		c.log.Warning(fmt.Sprintf("Attempt %d failed for cron job %s/%s: %v", attempt, handler.Group, handler.Name, lastError))

		if attempt <= retryLimit {
			// Calculate backoff if enabled
			delay := retryDelay
			if handler.UseExponentialBackoff {
				delay = CalculateBackoff(retryDelay, attempt, DefaultMaxBackoffDelay)
			}
			time.Sleep(time.Duration(delay) * time.Millisecond)
			c.updateMetrics(false, true) // Count retry
		}
	}

	// Log result
	if success {
		_, err = c.Log(job.Id, LogStatusSuccess, map[string]any{"result": result})
		if err != nil {
			c.log.Warning(fmt.Sprintf("Failed to log success for cron job %s/%s: %v", handler.Group, handler.Name, err))
		}
		c.updateMetrics(true, false)
	} else {
		_, err = c.Log(job.Id, LogStatusFailed, map[string]any{"error": lastError.Error()})
		if err != nil {
			c.log.Warning(fmt.Sprintf("Failed to log failure for cron job %s/%s: %v", handler.Group, handler.Name, err))
		}
	}
}

// executeJobWithTimeout executes the job handler with a timeout
func (c *Cron) executeJobWithTimeout(handler *CronHandler, job *CronJob) (any, error) {
	timeout := handler.TimeoutMs
	if timeout == 0 {
		timeout = DefaultJobTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	resultChan := make(chan struct {
		result any
		err    error
	}, 1)

	go func() {
		result, err := handler.Handler(ctx, job)
		resultChan <- struct {
			result any
			err    error
		}{result, err}
	}()

	select {
	case <-ctx.Done():
		return nil, errors.ErrCronJobTimeout
	case res := <-resultChan:
		return res.result, res.err
	}
}

// updateMetrics updates the metrics
func (c *Cron) updateMetrics(success bool, isRetry bool) {
	if c.metrics == nil {
		return
	}

	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	c.metrics.TotalRuns++
	if success {
		c.metrics.TotalSucceeded++
	} else if !isRetry {
		c.metrics.TotalFailed++
	}
	if isRetry {
		c.metrics.TotalRetried++
	}
}

// GetJob retrieves a cron job by group and name
func (c *Cron) GetJob(group, name string) (*CronJob, error) {
	var job CronJob
	err := c.db.DB.Where(`"group" = ? AND name = ?`, group, name).First(&job).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrCronJobNotFound
		}
		return nil, err
	}
	return &job, nil
}

// GetJobLogs retrieves execution logs for a cron job
func (c *Cron) GetJobLogs(jobId string, limit int) ([]CronLog, error) {
	if limit <= 0 {
		limit = 10
	}

	var logs []CronLog
	err := c.db.DB.Where("job_id = ? AND deleted_at IS NULL", jobId).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	return logs, err
}

// ListJobs lists all cron jobs, optionally filtered by group
func (c *Cron) ListJobs(group string) ([]CronJob, error) {
	var jobs []CronJob
	query := c.db.DB.Where("deleted_at IS NULL")

	if group != "" {
		query = query.Where(`"group" = ?`, group)
	}

	err := query.Order("priority DESC, created_at ASC").Find(&jobs).Error
	return jobs, err
}

// UpdateJobConfig updates the configuration of an existing cron job
func (c *Cron) UpdateJobConfig(group, name string, config *CronConfig) error {
	updates := map[string]any{
		"updated_at": time.Now().UTC(),
	}

	if config.Priority != 0 {
		updates["priority"] = config.Priority
	}
	if config.RetryLimit != 0 {
		updates["retry_limit"] = config.RetryLimit
	}
	if config.RetryDelay != 0 {
		delay := config.RetryDelay
		if delay > DefaultMaxRetryDelay {
			delay = DefaultMaxRetryDelay
		}
		updates["retry_delay"] = delay
	}
	if config.StartAt != nil {
		updates["start_at"] = config.StartAt
	}
	if config.ExpireAt != nil {
		updates["expire_at"] = config.ExpireAt
	}

	// Update config JSON
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	updates["config"] = configBytes

	result := c.db.DB.Model(&CronJob{}).
		Where(`"group" = ? AND name = ?`, group, name).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.ErrCronJobNotFound
	}

	return nil
}

// DisableJob disables a cron job (soft delete)
func (c *Cron) DisableJob(group, name string) error {
	now := time.Now().UTC()
	result := c.db.DB.Model(&CronJob{}).
		Where(`"group" = ? AND name = ? AND deleted_at IS NULL`, group, name).
		Updates(map[string]any{
			"deleted_at": &now,
			"updated_at": &now,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.ErrCronJobNotFound
	}

	return nil
}

// EnableJob re-enables a disabled cron job
func (c *Cron) EnableJob(group, name string) error {
	now := time.Now().UTC()
	result := c.db.DB.Model(&CronJob{}).
		Where(`"group" = ? AND name = ? AND deleted_at IS NOT NULL`, group, name).
		Updates(map[string]any{
			"deleted_at": nil,
			"status":     JobStatusPending,
			"updated_at": &now,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.ErrCronJobNotFound
	}

	return nil
}
