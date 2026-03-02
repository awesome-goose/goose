package cron

import (
	"strconv"
	"strings"
	"time"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/modules/sql"
)

// Standard cron format: minute hour day-of-month month day-of-week
// Example: "0 * * * *" = every hour at minute 0
// Example: "*/15 * * * *" = every 15 minutes
// Example: "0 9 * * 1-5" = 9 AM on weekdays

// IsValidCronPattern validates a cron expression
// Supports standard 5-field format: minute hour day month weekday
func IsValidCronPattern(pattern string) bool {
	fields := strings.Fields(pattern)
	if len(fields) != 5 {
		return false
	}

	// Validate each field
	validators := []struct {
		min, max int
	}{
		{0, 59}, // minute
		{0, 23}, // hour
		{1, 31}, // day of month
		{1, 12}, // month
		{0, 6},  // day of week (0 = Sunday)
	}

	for i, field := range fields {
		if !isValidCronField(field, validators[i].min, validators[i].max) {
			return false
		}
	}

	return true
}

// isValidCronField validates a single cron field
func isValidCronField(field string, min, max int) bool {
	// Handle wildcard
	if field == "*" {
		return true
	}

	// Handle step values: */n or m-n/s
	if strings.Contains(field, "/") {
		parts := strings.SplitN(field, "/", 2)
		if len(parts) != 2 {
			return false
		}
		// Validate the step value
		step, err := strconv.Atoi(parts[1])
		if err != nil || step < 1 {
			return false
		}
		// Validate the base (either * or a range)
		if parts[0] != "*" && !isValidCronField(parts[0], min, max) {
			return false
		}
		return true
	}

	// Handle list: a,b,c
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, part := range parts {
			if !isValidCronField(strings.TrimSpace(part), min, max) {
				return false
			}
		}
		return true
	}

	// Handle range: a-b
	if strings.Contains(field, "-") {
		parts := strings.SplitN(field, "-", 2)
		if len(parts) != 2 {
			return false
		}
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return false
		}
		return start >= min && start <= max && end >= min && end <= max && start <= end
	}

	// Handle single value
	val, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return val >= min && val <= max
}

// ShouldRunNow checks if a cron pattern matches the current time
// Uses a tolerance window for matching (default: 1 minute)
func ShouldRunNow(pattern string, now time.Time) bool {
	fields := strings.Fields(pattern)
	if len(fields) != 5 {
		return false
	}

	minute := now.Minute()
	hour := now.Hour()
	day := now.Day()
	month := int(now.Month())
	weekday := int(now.Weekday()) // 0 = Sunday

	// Check each field
	if !matchCronField(fields[0], minute, 0, 59) {
		return false
	}
	if !matchCronField(fields[1], hour, 0, 23) {
		return false
	}
	if !matchCronField(fields[2], day, 1, 31) {
		return false
	}
	if !matchCronField(fields[3], month, 1, 12) {
		return false
	}
	if !matchCronField(fields[4], weekday, 0, 6) {
		return false
	}

	return true
}

// matchCronField checks if a value matches a cron field expression
func matchCronField(field string, value, min, max int) bool {
	// Handle wildcard
	if field == "*" {
		return true
	}

	// Handle step values: */n
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step < 1 {
			return false
		}
		return value%step == 0
	}

	// Handle range step values: m-n/s
	if strings.Contains(field, "/") {
		parts := strings.SplitN(field, "/", 2)
		step, err := strconv.Atoi(parts[1])
		if err != nil || step < 1 {
			return false
		}

		// Get the range
		rangePart := parts[0]
		if rangePart == "*" {
			return value%step == 0
		}

		// Parse range
		rangeParts := strings.SplitN(rangePart, "-", 2)
		if len(rangeParts) != 2 {
			return false
		}
		start, _ := strconv.Atoi(rangeParts[0])
		end, _ := strconv.Atoi(rangeParts[1])

		if value < start || value > end {
			return false
		}
		return (value-start)%step == 0
	}

	// Handle list: a,b,c
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, part := range parts {
			if matchCronField(strings.TrimSpace(part), value, min, max) {
				return true
			}
		}
		return false
	}

	// Handle range: a-b
	if strings.Contains(field, "-") {
		parts := strings.SplitN(field, "-", 2)
		if len(parts) != 2 {
			return false
		}
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return false
		}
		return value >= start && value <= end
	}

	// Handle single value
	expected, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return value == expected
}

// GetNextRun calculates the next run time for a cron pattern
func GetNextRun(pattern string, from time.Time) (time.Time, error) {
	if !IsValidCronPattern(pattern) {
		return time.Time{}, errors.ErrCronInvalidPattern
	}

	// Start from the next minute
	next := from.Truncate(time.Minute).Add(time.Minute)

	// Search for up to 2 years
	maxIterations := 2 * 365 * 24 * 60 // 2 years in minutes

	for i := 0; i < maxIterations; i++ {
		if ShouldRunNow(pattern, next) {
			return next, nil
		}
		next = next.Add(time.Minute)
	}

	return time.Time{}, errors.ErrCronInvalidPattern
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

// startCleanup starts the background goroutine for cleaning up old logs
// and recovering stale jobs
func startCleanup(db *sql.Db, config *Config) {
	cleanupInterval := config.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = time.Hour
	}

	retentionPeriod := config.LogRetentionPeriod
	if retentionPeriod == 0 {
		retentionPeriod = 7 * 24 * time.Hour
	}

	staleTimeout := config.StaleJobTimeout
	if staleTimeout == 0 {
		staleTimeout = time.Duration(DefaultStaleJobTimeout) * time.Second
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UTC()
		cutoff := now.Add(-retentionPeriod)
		staleCutoff := now.Add(-staleTimeout)

		// Recover stale jobs (stuck in_progress for too long)
		if config.EnableStaleJobRecovery {
			db.DB.Model(&CronJob{}).
				Where("status = ? AND started_at IS NOT NULL AND started_at < ?",
					JobStatusInProgress, staleCutoff,
				).
				Updates(map[string]any{
					"status":     JobStatusPending,
					"started_at": nil,
					"updated_at": &now,
				})
		}

		// Clean up old logs (soft delete)
		db.DB.Model(&CronLog{}).
			Where("created_at < ? AND deleted_at IS NULL", cutoff).
			Updates(map[string]any{
				"deleted_at": &now,
				"updated_at": &now,
			})

		// Mark expired jobs
		db.DB.Model(&CronJob{}).
			Where("status = ? AND expire_at IS NOT NULL AND expire_at < ? AND deleted_at IS NULL",
				JobStatusPending, now,
			).
			Updates(map[string]any{
				"status":     JobStatusExpired,
				"expired_at": &now,
				"updated_at": &now,
			})
	}
}

// RecoverStaleJobs manually recovers jobs stuck in in_progress status
func RecoverStaleJobs(db *sql.Db, timeout time.Duration) (int64, error) {
	now := time.Now().UTC()
	staleCutoff := now.Add(-timeout)

	result := db.DB.Model(&CronJob{}).
		Where("status = ? AND started_at IS NOT NULL AND started_at < ?",
			JobStatusInProgress, staleCutoff,
		).
		Updates(map[string]any{
			"status":     JobStatusPending,
			"started_at": nil,
			"updated_at": &now,
		})

	return result.RowsAffected, result.Error
}

// Common cron patterns as constants for convenience
const (
	// PatternEveryMinute runs every minute "* * * * *"
	PatternEveryMinute = "* * * * *"
	// PatternEvery5Minutes runs every 5 minutes "*/5 * * * *"
	PatternEvery5Minutes = "*/5 * * * *"
	// PatternEvery15Minutes runs every 15 minutes "*/15 * * * *"
	PatternEvery15Minutes = "*/15 * * * *"
	// PatternEvery30Minutes runs every 30 minutes "*/30 * * * *"
	PatternEvery30Minutes = "*/30 * * * *"
	// PatternEveryHour runs at the start of every hour "0 * * * *"
	PatternEveryHour = "0 * * * *"
	// PatternEveryDay runs at midnight every day "0 0 * * *"
	PatternEveryDay = "0 0 * * *"
	// PatternEveryWeek runs at midnight on Sundays "0 0 * * 0"
	PatternEveryWeek = "0 0 * * 0"
	// PatternEveryMonth runs at midnight on the 1st of each month "0 0 1 * *"
	PatternEveryMonth = "0 0 1 * *"
	// PatternWeekdays runs at midnight on weekdays "0 0 * * 1-5"
	PatternWeekdays = "0 0 * * 1-5"
	// PatternWeekends runs at midnight on weekends "0 0 * * 0,6"
	PatternWeekends = "0 0 * * 0,6"
)
