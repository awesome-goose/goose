package kv

import (
	"time"

	"github.com/awesome-goose/goose/modules/sql"
)

// globToSQL converts a glob pattern to SQL LIKE pattern
func globToSQL(pattern string) string {
	result := ""
	for _, c := range pattern {
		switch c {
		case '*':
			result += "%"
		case '?':
			result += "_"
		case '%', '_':
			result += "\\" + string(c)
		default:
			result += string(c)
		}
	}
	return result
}

// startCleanup starts the background goroutine for cleaning up expired keys
func startCleanup(db *sql.Db, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UTC()
		db.DB.Model(&KVStore{}).
			Where("expired_at < ? AND status = ?", now, StatusActive).
			Updates(map[string]any{
				"status":     StatusDeleted,
				"deleted_at": &now,
				"updated_at": &now,
			})
	}
}
