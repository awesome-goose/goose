package sql

import (
	"strconv"
	"strings"
)

// defaultPostgresSchema is where Postgres puts objects when no schema is set.
const defaultPostgresSchema = "public"

// PostgresDSN returns the libpq keyword/value connection string for config.
//
// Every value except TimeZone is single-quoted and escaped, so an empty,
// spaced or quoted value (a blank password, `p ss'word`) cannot run into the
// next key. Fields that are unset are left out so the driver applies its own
// defaults.
//
// TimeZone is deliberately NOT quoted: gorm.io/driver/postgres.Initialize
// re-parses the raw DSN with its own regexp (`timeZoneMatcher`) to pull
// TimeZone out as a startup runtime parameter, ahead of and separately from
// pgx's own (correct) DSN parsing, and that regexp knows nothing about libpq
// quoting — it captures everything up to the next space or `&` verbatim, quote
// characters included, which Postgres then rejects outright ("invalid value
// for parameter TimeZone"). No valid IANA zone name contains a quote, space or
// `&`, so leaving it unquoted is safe; a value that did contain one is skipped
// rather than sent, since gorm's regexp would silently truncate it at that
// character either way.
func PostgresDSN(c *Config) string {
	var parts []string
	add := func(key, value string) {
		if value != "" {
			parts = append(parts, key+"="+quoteDSNValue(value))
		}
	}

	add("host", c.Host)
	add("user", c.User)
	add("password", c.Pass)
	add("dbname", c.Name)
	if c.Port != 0 {
		add("port", strconv.Itoa(c.Port))
	}
	add("sslmode", c.SSLMode)
	add("search_path", PostgresSchema(c))
	if tz := c.TimeZone; tz != "" && !strings.ContainsAny(tz, " '\"&") {
		parts = append(parts, "TimeZone="+tz)
	}

	return strings.Join(parts, " ")
}

// PostgresSchema returns the schema the module's tables live in: the configured
// one, or "public" when none is set.
func PostgresSchema(c *Config) string {
	if c.Schema == "" {
		return defaultPostgresSchema
	}
	return c.Schema
}

// quoteDSNValue quotes a libpq keyword/value value: single quotes around it, with
// backslash and single quote escaped by a backslash.
func quoteDSNValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}
