package sql

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"
)

// BUGS #23: with Config.TimeZone set, a timestamptz comes back in that zone, not in the
// machine's. Needs a real Postgres (GOOSE_TEST_DB_HOST, optionally _PORT/_USER/_PASS/_NAME).
func TestTimestamptzScansInTheConfiguredZone(t *testing.T) {
	host := os.Getenv("GOOSE_TEST_DB_HOST")
	if host == "" {
		t.Skip("GOOSE_TEST_DB_HOST not set — skipping (needs a real Postgres)")
	}
	port := 5432
	if p := os.Getenv("GOOSE_TEST_DB_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}
	cfg := func(tz string) *Config {
		name := os.Getenv("GOOSE_TEST_DB_NAME")
		if name == "" {
			name = "postgres"
		}
		user := os.Getenv("GOOSE_TEST_DB_USER")
		if user == "" {
			user = "postgres"
		}
		return &Config{Dialect: "postgres", Host: host, Port: port, User: user, Pass: os.Getenv("GOOSE_TEST_DB_PASS"), Name: name, SSLMode: "disable", TimeZone: tz}
	}
	// A machine one hour east of UTC, like the one the bug was seen on.
	saved := time.Local
	time.Local = time.FixedZone("WAT", 3600)
	defer func() { time.Local = saved }()

	scan := func(c *Config) time.Time {
		db, err := gorm.Open(postgresDialector(PostgresDSN(c), c.TimeZone), &gorm.Config{})
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()
		var at time.Time
		if err := db.Raw("SELECT TIMESTAMPTZ '2026-11-10 18:28:12.5+00'").Scan(&at).Error; err != nil {
			t.Fatalf("scan: %v", err)
		}
		return at
	}

	utc := scan(cfg("UTC"))
	if utc.Location() != time.UTC || utc.Format(time.RFC3339Nano) != "2026-11-10T18:28:12.5Z" {
		t.Fatalf("TimeZone=UTC scanned %s (%s), want 2026-11-10T18:28:12.5Z", utc.Format(time.RFC3339Nano), utc.Location())
	}
	local := scan(cfg(""))
	if !local.Equal(utc) || local.Location() == time.UTC {
		t.Fatalf("no TimeZone: want the same instant in machine time (as before), got %s", local.Format(time.RFC3339Nano))
	}
}
