package cache

import (
	"time"

	"github.com/awesome-goose/goose/modules/sql"
)

// startCleanup starts the background goroutine for cleaning up expired cache entries
func startCleanup(db *sql.Db, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UTC()
		db.DB.Model(&CacheStore{}).
			Where("expired_at < ? AND status = ?", now, StatusActive).
			Updates(map[string]any{
				"status":     StatusDeleted,
				"deleted_at": &now,
				"updated_at": &now,
			})
	}
}
