package queues

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/awesome-goose/goose/modules/sql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// failures records every query GORM reports as failed (what its default
// logger prints in red).
type failures struct {
	mu   sync.Mutex
	errs []error
}

func (f *failures) LogMode(logger.LogLevel) logger.Interface      { return f }
func (f *failures) Info(context.Context, string, ...interface{})  {}
func (f *failures) Warn(context.Context, string, ...interface{})  {}
func (f *failures) Error(context.Context, string, ...interface{}) {}
func (f *failures) Trace(_ context.Context, _ time.Time, _ func() (string, int64), err error) {
	if err != nil {
		f.mu.Lock()
		f.errs = append(f.errs, err)
		f.mu.Unlock()
	}
}

// A worker polls Pop every few seconds, usually before anything was ever
// pushed to its queue. That is the normal idle state: Pop answers nil, nil
// and nothing is logged as a failed query (each poll used to log "record not
// found").
func TestPopOnAQueueNothingWasPushedToIsQuiet(t *testing.T) {
	f := &failures{}
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: f})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&QueueQueue{}, &QueueJob{}); err != nil {
		t.Fatal(err)
	}
	f.errs = nil
	q := &Queue{db: &sql.Db{DB: db}}

	job, err := q.Pop("queue/never-pushed", "Jobs")
	if err != nil || job != nil {
		t.Fatalf("Pop on an empty queue: job %v, err %v", job, err)
	}
	for _, e := range f.errs {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			t.Fatalf("an idle poll was logged as a failed query: %v", e)
		}
	}
}
