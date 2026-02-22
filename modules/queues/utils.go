package queues

import (
	"time"

	"github.com/awesome-goose/goose/modules/sql"
)

// startCleanup starts the background goroutine for cleaning up old completed/failed jobs
// and recovering stale jobs that have been stuck in_progress
func startCleanup(db *sql.Db, interval time.Duration, retentionPeriod time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Default stale job timeout (5 minutes)
	staleTimeout := time.Duration(DefaultStaleJobTimeout) * time.Second

	for range ticker.C {
		now := time.Now().UTC()
		cutoff := now.Add(-retentionPeriod)
		staleCutoff := now.Add(-staleTimeout)

		// Recover stale jobs (stuck in_progress for too long)
		// These jobs likely belong to workers that crashed
		db.DB.Model(&QueueJob{}).
			Where("status = ? AND started_at IS NOT NULL AND started_at < ?",
				JobStatusInProgress, staleCutoff,
			).
			Updates(map[string]any{
				"status":     JobStatusNew,
				"started_at": nil,
				"updated_at": &now,
			})

		// Clean up expired jobs (soft delete)
		db.DB.Model(&QueueJob{}).
			Where("status IN ? AND updated_at < ? AND deleted_at IS NULL",
				[]string{JobStatusSuccess, JobStatusFailed, JobStatusExpired, JobStatusTimeout},
				cutoff,
			).
			Updates(map[string]any{
				"deleted_at": &now,
				"updated_at": &now,
			})

		// Clean up old logs (soft delete)
		db.DB.Model(&QueueLog{}).
			Where("created_at < ? AND deleted_at IS NULL", cutoff).
			Updates(map[string]any{
				"deleted_at": &now,
				"updated_at": &now,
			})

		// Mark expired jobs (past expire_at)
		db.DB.Model(&QueueJob{}).
			Where("status = ? AND expire_at IS NOT NULL AND expire_at < ?",
				JobStatusNew, now,
			).
			Updates(map[string]any{
				"status":     JobStatusExpired,
				"expired_at": &now,
				"updated_at": &now,
			})
	}
}

// RecoverStaleJobs manually recovers jobs stuck in in_progress status
// This can be called on startup or periodically
func RecoverStaleJobs(db *sql.Db, timeout time.Duration) (int64, error) {
	now := time.Now().UTC()
	staleCutoff := now.Add(-timeout)

	result := db.DB.Model(&QueueJob{}).
		Where("status = ? AND started_at IS NOT NULL AND started_at < ?",
			JobStatusInProgress, staleCutoff,
		).
		Updates(map[string]any{
			"status":     JobStatusNew,
			"started_at": nil,
			"updated_at": &now,
		})

	return result.RowsAffected, result.Error
}

// CalculateBackoff calculates exponential backoff delay
func CalculateBackoff(baseDelay, attempt, maxDelay int) int {
	if attempt <= 0 {
		return baseDelay
	}
	delay := baseDelay * (1 << (attempt - 1)) // 2^(attempt-1) * base
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}
