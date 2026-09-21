package tests

import (
	"testing"

	"github.com/awesome-goose/goose/modules/sql"
	test "github.com/awesome-goose/goose/testing"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSqlPostgresDSN(t *testing.T) {
	test.NewSuiteRunner(t, &SqlPostgresDSNSuite{}).Run()
}

type SqlPostgresDSNSuite struct {
	test.Suite
}

func baseSqlConfig() *sql.Config {
	return &sql.Config{
		Host:     "db.internal",
		Port:     6543,
		User:     "app",
		Pass:     "secret",
		Name:     "appdb",
		SSLMode:  "disable",
		Schema:   "tenant",
		TimeZone: "UTC",
	}
}

// parse reads the DSN the way the postgres driver does.
func (s *SqlPostgresDSNSuite) parse(c *sql.Config) *pgconn.Config {
	parsed, err := pgconn.ParseConfig(sql.PostgresDSN(c))
	s.T.Require(err).ToBeNil()
	return parsed
}

func (s *SqlPostgresDSNSuite) TestAllFieldsReachTheDriver() {
	parsed := s.parse(baseSqlConfig())

	s.T.Expect(parsed.Host).ToEqual("db.internal")
	s.T.Expect(parsed.Port).ToEqual(uint16(6543))
	s.T.Expect(parsed.User).ToEqual("app")
	s.T.Expect(parsed.Password).ToEqual("secret")
	s.T.Expect(parsed.Database).ToEqual("appdb")
	s.T.Expect(parsed.RuntimeParams["search_path"]).ToEqual("tenant")
	s.T.Expect(parsed.RuntimeParams["TimeZone"]).ToEqual("UTC")
}

// An empty password used to swallow the next key: `password= dbname=x` made the
// driver read "dbname=x" as the password and connect with an empty database name.
func (s *SqlPostgresDSNSuite) TestEmptyPasswordDoesNotSwallowTheDatabaseName() {
	c := baseSqlConfig()
	c.Pass = ""

	parsed := s.parse(c)

	s.T.Expect(parsed.Database).ToEqual("appdb")
	s.T.Expect(parsed.Password).ToEqual("")
	s.T.Expect(parsed.Port).ToEqual(uint16(6543))
}

func (s *SqlPostgresDSNSuite) TestValuesWithSpacesQuotesAndBackslashesSurvive() {
	c := baseSqlConfig()
	c.Pass = `p ss'w\rd dbname=evil`
	c.User = "app user"

	parsed := s.parse(c)

	s.T.Expect(parsed.Password).ToEqual(`p ss'w\rd dbname=evil`)
	s.T.Expect(parsed.User).ToEqual("app user")
	s.T.Expect(parsed.Database).ToEqual("appdb")
}

func (s *SqlPostgresDSNSuite) TestEmptyOptionalFieldsAreOmittedNotSentEmpty() {
	c := baseSqlConfig()
	c.SSLMode = ""
	c.TimeZone = ""

	parsed := s.parse(c)

	_, hasTimeZone := parsed.RuntimeParams["TimeZone"]
	s.T.Expect(hasTimeZone).ToEqual(false)
	s.T.Expect(parsed.Database).ToEqual("appdb")
}

func (s *SqlPostgresDSNSuite) TestEmptySchemaDefaultsToPublic() {
	c := baseSqlConfig()
	c.Schema = ""

	s.T.Expect(sql.PostgresSchema(c)).ToEqual("public")
	s.T.Expect(s.parse(c).RuntimeParams["search_path"]).ToEqual("public")
}

func (s *SqlPostgresDSNSuite) TestConfiguredSchemaIsKept() {
	s.T.Expect(sql.PostgresSchema(baseSqlConfig())).ToEqual("tenant")
}
