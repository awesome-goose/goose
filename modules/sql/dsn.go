package sql

import (
	"strconv"
	"strings"
)

// defaultPostgresSchema is where Postgres puts objects when no schema is set.
const defaultPostgresSchema = "public"

// PostgresDSN returns the libpq keyword/value connection string for config.
//
// Every value is single-quoted and escaped, so an empty, spaced or quoted value
// (a blank password, `p ss'word`) cannot run into the next key. Fields that are
// unset are left out so the driver applies its own defaults.
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
	add("TimeZone", c.TimeZone)

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
