package tests

import (
	"os"
	"testing"

	"github.com/awesome-goose/goose/modules/sql"
	test "github.com/awesome-goose/goose/testing"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestSqlPostgresDSNThroughGorm(t *testing.T) {
	test.NewSuiteRunner(t, &SqlPostgresDSNThroughGormSuite{}).Run()
}

// Real Postgres, not a mock: gorm.io/driver/postgres.Initialize() re-parses the
// DSN with its own regexp (`timeZoneMatcher`) to pull out TimeZone as a runtime
// startup parameter — separately from, and less carefully than, pgx's own DSN
// parser (which sql_dsn_test.go exercises against pgconn.ParseConfig alone,
// never touching gorm). That second, cruder parser is exactly what
// PostgresDSN's quoting has to stay compatible with, and only an actual
// connection proves it: a value it mishandles fails the connection outright
// ("invalid value for parameter"), it doesn't silently misbehave.
//
// Needs a real, reachable Postgres — set GOOSE_TEST_DB_HOST (and optionally
// _PORT/_USER/_NAME); skips itself otherwise.
type SqlPostgresDSNThroughGormSuite struct {
	test.Suite
}

func (s *SqlPostgresDSNThroughGormSuite) baseConfig() *sql.Config {
	host := os.Getenv("GOOSE_TEST_DB_HOST")
	if host == "" {
		s.T.T().Skip("GOOSE_TEST_DB_HOST not set — skipping (needs a real Postgres)")
	}
	return &sql.Config{
		Dialect: "postgres",
		Host:    host,
		Port:    envInt("GOOSE_TEST_DB_PORT", 5432),
		User:    envOr("GOOSE_TEST_DB_USER", "root"),
		Pass:    os.Getenv("GOOSE_TEST_DB_PASS"),
		Name:    envOr("GOOSE_TEST_DB_NAME", "goose_dsn_test"),
		SSLMode: "disable",
	}
}

func (s *SqlPostgresDSNThroughGormSuite) connect(c *sql.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(sql.PostgresDSN(c)), &gorm.Config{})
}

func (s *SqlPostgresDSNThroughGormSuite) showTimeZone(db *gorm.DB) string {
	var tz string
	s.T.Require(db.Raw("SHOW timezone").Scan(&tz).Error).ToBeNil()
	return tz
}

func (s *SqlPostgresDSNThroughGormSuite) TestConnectsWithUTC() {
	c := s.baseConfig()
	c.TimeZone = "UTC"

	db, err := s.connect(c)
	s.T.Require(err).ToBeNil()
	s.T.Expect(s.showTimeZone(db)).ToEqual("UTC")
}

func (s *SqlPostgresDSNThroughGormSuite) TestConnectsWithANonUTCTimeZone() {
	c := s.baseConfig()
	c.TimeZone = "America/New_York"

	db, err := s.connect(c)
	s.T.Require(err).ToBeNil()
	s.T.Expect(s.showTimeZone(db)).ToEqual("America/New_York")
}

func (s *SqlPostgresDSNThroughGormSuite) TestConnectsWithNoTimeZoneConfigured() {
	c := s.baseConfig()
	c.TimeZone = ""

	_, err := s.connect(c)
	s.T.Expect(err).ToBeNil()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}
