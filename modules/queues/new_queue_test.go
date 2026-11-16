package queues

import (
	"context"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awesome-goose/goose/modules/sql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// A Queue built outside the container pushes, processes and records jobs.
// Needs a real Postgres (GOOSE_TEST_DB_HOST, optionally _PORT/_USER/_PASS/_NAME).
func TestNewQueuePushAndProcess(t *testing.T) {
	host := os.Getenv("GOOSE_TEST_DB_HOST")
	if host == "" {
		t.Skip("GOOSE_TEST_DB_HOST not set — skipping (needs a real Postgres)")
	}
	port, _ := strconv.Atoi(os.Getenv("GOOSE_TEST_DB_PORT"))
	if port == 0 {
		port = 5432
	}
	name, user := os.Getenv("GOOSE_TEST_DB_NAME"), os.Getenv("GOOSE_TEST_DB_USER")
	if name == "" {
		name = "postgres"
	}
	if user == "" {
		user = "postgres"
	}
	c := &sql.Config{Dialect: "postgres", Host: host, Port: port, User: user, Pass: os.Getenv("GOOSE_TEST_DB_PASS"), Name: name, SSLMode: "disable"}
	db, err := gorm.Open(postgres.Open(sql.PostgresDSN(c)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tbl := range []string{"QueueLogs", "QueueJobs", "QueueQueues"} {
		db.Exec(`DROP TABLE IF EXISTS "` + tbl + `" CASCADE`)
	}
	if err := db.AutoMigrate(&QueueQueue{}, &QueueJob{}, &QueueLog{}); err != nil {
		t.Fatal(err)
	}
	q := NewQueue(db, nil)
	defer q.Shutdown()
	if _, err := q.Push("q1", "work", map[string]int{"n": 1}, nil); err != nil {
		t.Fatal(err)
	}
	var ran atomic.Int32
	q.Process(NewHandler("q1", "work", func(ctx context.Context, j *QueueJob) (any, error) {
		ran.Add(1)
		return nil, nil
	}).WithPollInterval(20))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		jobs, _ := q.ListJobs("q1", JobStatusSuccess, 10)
		if len(jobs) == 1 {
			if ran.Load() != 1 {
				t.Fatalf("handler ran %d times", ran.Load())
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("job did not succeed")
}
