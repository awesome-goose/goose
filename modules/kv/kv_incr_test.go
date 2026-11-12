package kv

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"testing"

	"github.com/awesome-goose/goose/modules/kv/migrations"
	"github.com/awesome-goose/goose/modules/sql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// U-G11: IncrBy read the row, added in Go and wrote it back with no lock, and
// (group, key) was not unique, so concurrent increments were lost and two
// first increments could each create a row. These tests need a real Postgres:
// set GOOSE_TEST_DB_HOST (and GOOSE_TEST_DB_PORT, _USER, _PASS, _NAME).

const kvHelperEnv = "GOOSE_KV_INCR_HELPER"

func kvPostgresDSN(t testing.TB) string {
	host := os.Getenv("GOOSE_TEST_DB_HOST")
	if host == "" {
		t.Skip("GOOSE_TEST_DB_HOST not set — skipping (needs a real Postgres)")
	}
	port, _ := strconv.Atoi(kvEnvOr("GOOSE_TEST_DB_PORT", "5432"))
	return sql.PostgresDSN(&sql.Config{
		Dialect: "postgres",
		Host:    host,
		Port:    port,
		User:    kvEnvOr("GOOSE_TEST_DB_USER", "postgres"),
		Pass:    os.Getenv("GOOSE_TEST_DB_PASS"),
		Name:    kvEnvOr("GOOSE_TEST_DB_NAME", "goose_kv_test"),
	})
}

func kvEnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func openKV(t testing.TB, dsn string, group string) *KV {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(20)
	return &KV{db: &sql.Db{DB: db}, config: &Config{Group: group}}
}

// freshKV drops the table and migrations record, then runs the migrations.
func freshKV(t *testing.T, group string) (*KV, string) {
	dsn := kvPostgresDSN(t)
	kv := openKV(t, dsn, group)
	kv.db.DB.Exec(`DROP TABLE IF EXISTS "KVStore"`)
	kv.db.DB.Exec(`DELETE FROM "Migrations" WHERE name LIKE '%KVStore%' OR name LIKE 'kv/%'`)
	if err := sql.Migrate(kv.db.DB, migrations.Migrations()); err != nil {
		t.Fatal(err)
	}
	return kv, dsn
}

func TestKVIncrByConcurrentGoroutines(t *testing.T) {
	kv, _ := freshKV(t, "incr-goroutines")

	const workers, each = 40, 25
	seen := make(chan int64, workers*each)
	var wg sync.WaitGroup
	errs := make(chan error, workers*each)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				n, err := kv.IncrBy("counter", 1)
				if err != nil {
					errs <- err
					return
				}
				seen <- n
			}
		}()
	}
	wg.Wait()
	close(seen)
	close(errs)
	for err := range errs {
		t.Fatalf("IncrBy: %v", err)
	}

	// Each increment returns its own value: 1..workers*each, none twice.
	got := map[int64]bool{}
	for n := range seen {
		if got[n] {
			t.Fatalf("value %d returned twice", n)
		}
		got[n] = true
	}
	if len(got) != workers*each {
		t.Fatalf("got %d distinct values, want %d", len(got), workers*each)
	}
	v, err := kv.Get("counter")
	if err != nil {
		t.Fatal(err)
	}
	if int64(v.(float64)) != workers*each {
		t.Fatalf("counter = %v, want %d", v, workers*each)
	}
	var rows int64
	kv.db.DB.Model(&KVStore{}).Where(`"group" = ? AND "key" = ?`, "incr-goroutines", "counter").Count(&rows)
	if rows != 1 {
		t.Fatalf("%d rows for one key, want 1", rows)
	}
}

// TestKVIncrByHelperProcess is the body each child process runs.
func TestKVIncrByHelperProcess(t *testing.T) {
	if os.Getenv(kvHelperEnv) != "1" {
		t.Skip("helper process")
	}
	kv := openKV(t, kvPostgresDSN(t), "incr-processes")
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				if _, err := kv.IncrBy("counter", 2); err != nil {
					t.Errorf("IncrBy: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestKVIncrByConcurrentProcesses(t *testing.T) {
	kv, _ := freshKV(t, "incr-processes")

	const procs = 4
	var wg sync.WaitGroup
	outs := make([]string, procs)
	fails := make([]error, procs)
	for p := 0; p < procs; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			cmd := exec.Command(os.Args[0], "-test.run=^TestKVIncrByHelperProcess$", "-test.count=1")
			cmd.Env = append(os.Environ(), kvHelperEnv+"=1")
			out, err := cmd.CombinedOutput()
			outs[p], fails[p] = string(out), err
		}(p)
	}
	wg.Wait()
	for p := range fails {
		if fails[p] != nil {
			t.Fatalf("process %d: %v\n%s", p, fails[p], outs[p])
		}
	}

	v, err := kv.Get("counter")
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(procs * 8 * 25 * 2); int64(v.(float64)) != want {
		t.Fatalf("counter = %v, want %d", v, want)
	}
}

func TestKVGroupKeyIsUnique(t *testing.T) {
	kv, _ := freshKV(t, "unique")
	if err := kv.Set("k", 1); err != nil {
		t.Fatal(err)
	}
	dup := &KVStore{Group: "unique", Key: "k", Value: []byte("2"), Status: StatusActive}
	if err := kv.db.DB.Create(dup).Error; err == nil {
		t.Fatal("a second row for the same (group, key) was accepted")
	}
}

// A deleted or expired key counts from zero, and SetNX/GetSet/Set reuse the
// one row the unique index allows.
func TestKVAfterDeleteReusesTheRow(t *testing.T) {
	kv, _ := freshKV(t, "reuse")
	if _, err := kv.IncrBy("n", 5); err != nil {
		t.Fatal(err)
	}
	if _, err := kv.Del("n"); err != nil {
		t.Fatal(err)
	}
	n, err := kv.IncrBy("n", 3)
	if err != nil || n != 3 {
		t.Fatalf("IncrBy after Del = %d, %v; want 3", n, err)
	}

	kv.Del("n")
	ok, err := kv.SetNX("n", "x")
	if err != nil || !ok {
		t.Fatalf("SetNX after Del = %v, %v; want true", ok, err)
	}
	ok, err = kv.SetNX("n", "y")
	if err != nil || ok {
		t.Fatalf("SetNX on a live key = %v, %v; want false", ok, err)
	}

	kv.Del("n")
	old, err := kv.GetSet("n", "z")
	if err != nil || old != nil {
		t.Fatalf("GetSet after Del = %v, %v; want nil", old, err)
	}
	kv.Del("n")
	if err := kv.Set("n", "w"); err != nil {
		t.Fatal(err)
	}
	if v, _ := kv.Get("n"); v != "w" {
		t.Fatalf("Get = %v, want w", v)
	}
}

// Concurrent SetNX of a missing key: exactly one wins, none errors.
func TestKVSetNXConcurrent(t *testing.T) {
	kv, _ := freshKV(t, "setnx")
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ok, err := kv.SetNX("lock", i)
			if err != nil {
				t.Errorf("SetNX: %v", err)
				return
			}
			if ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("%d SetNX calls won, want 1", wins)
	}
}

// A database that already holds duplicate rows (from the old racy IncrBy)
// still migrates: the newest active row is kept.
func TestKVUniqueMigrationKeepsNewestDuplicate(t *testing.T) {
	dsn := kvPostgresDSN(t)
	kv := openKV(t, dsn, "dups")
	kv.db.DB.Exec(`DROP TABLE IF EXISTS "KVStore"`)
	kv.db.DB.Exec(`DELETE FROM "Migrations" WHERE name LIKE '%KVStore%' OR name LIKE 'kv/%'`)
	ms := migrations.Migrations()
	if err := sql.Migrate(kv.db.DB, ms[:1]); err != nil {
		t.Fatal(err)
	}
	for i, ts := range []string{"2026-01-01", "2026-03-01", "2026-02-01"} {
		if err := kv.db.DB.Exec(`INSERT INTO "KVStore" (id, created_at, updated_at, "group", "key", value, status)
			VALUES (gen_random_uuid(), ?, ?, 'dups', 'k', ?, 'active')`, ts, ts, fmt.Sprint(i)).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := sql.Migrate(kv.db.DB, ms); err != nil {
		t.Fatal(err)
	}
	var rows int64
	kv.db.DB.Model(&KVStore{}).Where(`"group" = 'dups'`).Count(&rows)
	if rows != 1 {
		t.Fatalf("%d rows after the migration, want 1", rows)
	}
	if v, _ := kv.Get("k"); v != float64(1) {
		t.Fatalf("kept %v, want the newest (1)", v)
	}
}

// SQLite: the same calls work (no FOR UPDATE; SQLite serializes writers).
func TestKVIncrBySQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.TempDir()+"/kv.db?_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	// The schema declares TIMESTAMP(6), which the sqlite3 driver does not
	// parse back into time.Time, so this table uses DATETIME.
	if err := db.Exec(`CREATE TABLE "KVStore" (id TEXT PRIMARY KEY, created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL, deleted_at DATETIME, "group" TEXT NOT NULL, "key" TEXT NOT NULL,
		value BLOB, expired_at DATETIME, meta BLOB, status TEXT NOT NULL DEFAULT 'active')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := sql.Migrate(db, migrations.Migrations()[1:]); err != nil {
		t.Fatal(err)
	}
	kv := &KV{db: &sql.Db{DB: db}, config: &Config{Group: "lite"}}
	for i := 1; i <= 5; i++ {
		n, err := kv.IncrBy("c", 1)
		if err != nil || n != int64(i) {
			t.Fatalf("IncrBy = %d, %v; want %d", n, err, i)
		}
	}
	kv.Del("c")
	if n, err := kv.IncrBy("c", 2); err != nil || n != 2 {
		t.Fatalf("IncrBy after Del = %d, %v", n, err)
	}
	if ok, err := kv.SetNX("s", 1); err != nil || !ok {
		t.Fatalf("SetNX = %v, %v", ok, err)
	}
}
