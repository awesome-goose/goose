package sql

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/awesome-goose/goose/errors"
)

// JSONB helpers (Postgres). They build WHERE clauses and index DDL for jsonb
// columns in the form Entity and Query take: a clause with "?" placeholders
// and its args. Column names and path keys are checked and written into the
// clause; values are always bound, never written in.
//
//	where, args, _ := sql.JSONEq("meta", "priority", "high")       // meta->>'priority' = ?
//	rows, err := tasks.ForTenant(ws).Find(0, 50, where, args...)
//
//	where, args, _ = sql.JSONContains("labels", []string{"urgent"}) // labels @> ?::jsonb
//
// A migration adds the index that makes @> and key lookups fast:
//
//	func (m *AddTaskMetaIndex) Run(q *sql.Query) error {
//		return sql.CreateGINIndex(q, "Tasks", "meta")
//	}

// validJSONKey is what a path key may contain. Keys are written into the
// clause as string literals, so the set is deliberately small: no quote,
// backslash, brace or comma can appear.
var validJSONKey = regexp.MustCompile(`^[A-Za-z0-9_\-. ]+$`)

// JSONText is the SQL expression for the text at path in a jsonb column: one
// key gives "column"->>'key', several give "column" #>> '{a,b}'. Use it in a
// WHERE clause, an ORDER BY, or an expression index.
func JSONText(column string, path ...string) (string, error) {
	col, err := quoteColumn(column)
	if err != nil {
		return "", err
	}
	if len(path) == 0 {
		return "", errors.ErrInvalidJSONPath.WithMeta(column)
	}
	for _, k := range path {
		if !validJSONKey.MatchString(k) {
			return "", errors.ErrInvalidJSONPath.WithMeta(k)
		}
	}
	if len(path) == 1 {
		return col + "->>'" + path[0] + "'", nil
	}
	return col + " #>> '{" + strings.Join(quotePathKeys(path), ",") + "}'", nil
}

// JSONEq matches rows whose jsonb column has text value at key:
// "column"->>'key' = ?. For a nested path use JSONPathEq.
func JSONEq(column, key string, value any) (string, []any, error) {
	return JSONPathEq(column, []string{key}, value)
}

// JSONPathEq matches rows whose jsonb column has text value at path:
// "column" #>> '{a,b}' = ?. The comparison is on text, as ->> returns: pass
// "42" or 42 for a JSON number 42, "true" or true for a JSON boolean.
func JSONPathEq(column string, path []string, value any) (string, []any, error) {
	expr, err := JSONText(column, path...)
	if err != nil {
		return "", nil, err
	}
	return expr + " = ?", []any{jsonText(value)}, nil
}

// JSONContains matches rows whose jsonb column contains value ("column" @> ?):
// an object contains another when every key of the other is in it with a
// contained value, an array contains another when every element is in it.
// value is marshalled to JSON and bound. A GIN index serves it.
func JSONContains(column string, value any) (string, []any, error) {
	col, err := quoteColumn(column)
	if err != nil {
		return "", nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", nil, err
	}
	return col + " @> ?::jsonb", []any{string(raw)}, nil
}

// JSONHasKey matches rows whose jsonb column has key at its top level (or, for
// an array column, has the string as an element). It is Postgres's "?"
// operator, written as jsonb_exists so it cannot be taken for a placeholder.
func JSONHasKey(column, key string) (string, []any, error) {
	col, err := quoteColumn(column)
	if err != nil {
		return "", nil, err
	}
	return "jsonb_exists(" + col + ", ?)", []any{key}, nil
}

// JSONHasAnyKey matches rows whose jsonb column has at least one of keys at
// its top level (Postgres's "?|"). No keys matches nothing.
func JSONHasAnyKey(column string, keys ...string) (string, []any, error) {
	col, err := quoteColumn(column)
	if err != nil {
		return "", nil, err
	}
	if len(keys) == 0 {
		return "1 = 0", nil, nil
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(keys)), ",")
	args := make([]any, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	return "jsonb_exists_any(" + col + ", ARRAY[" + marks + "]::text[])", args, nil
}

// GINOption changes the index CreateGINIndex creates.
type GINOption func(*ginIndex)

type ginIndex struct {
	name    string
	pathOps bool
	path    []string
}

// GINName names the index, instead of idx_<table>_<column>_gin.
func GINName(name string) GINOption { return func(g *ginIndex) { g.name = name } }

// GINPathOps uses the jsonb_path_ops operator class: a smaller, faster index
// that serves @> (JSONContains) only, not key-exists lookups.
func GINPathOps() GINOption { return func(g *ginIndex) { g.pathOps = true } }

// CreateGINIndex creates a GIN index on a jsonb column, if it does not exist:
//
//	CREATE INDEX IF NOT EXISTS "idx_Tasks_meta_gin" ON "Tasks" USING GIN ("meta")
//
// Run it from a Migration. It serves JSONContains, JSONHasKey and
// JSONHasAnyKey. For equality on one path (JSONEq), CreateJSONPathIndex is the
// better index.
func CreateGINIndex(q *Query, table, column string, opts ...GINOption) error {
	stmt, err := ginIndexSQL(table, column, opts...)
	if err != nil {
		return err
	}
	_, err = q.Exec(stmt)
	return err
}

// CreateJSONPathIndex creates a b-tree index on the text at path in a jsonb
// column, if it does not exist: the index JSONEq and JSONPathEq use.
//
//	CREATE INDEX IF NOT EXISTS "idx_Tasks_meta_priority" ON "Tasks" (("meta"->>'priority'))
func CreateJSONPathIndex(q *Query, table, column string, path []string, opts ...GINOption) error {
	stmt, err := jsonPathIndexSQL(table, column, path, opts...)
	if err != nil {
		return err
	}
	_, err = q.Exec(stmt)
	return err
}

// DropIndex drops an index by name, if it exists (a migration's way back).
func DropIndex(q *Query, name string) error {
	if !validIdentifier.MatchString(name) || strings.Contains(name, ".") {
		return errors.ErrInvalidColumnName.WithMeta(name)
	}
	_, err := q.Exec(`DROP INDEX IF EXISTS "` + name + `"`)
	return err
}

func ginIndexSQL(table, column string, opts ...GINOption) (string, error) {
	g, tbl, col, err := indexParts(table, column, opts)
	if err != nil {
		return "", err
	}
	name := g.name
	if name == "" {
		name = "idx_" + table + "_" + column + "_gin"
	}
	class := ""
	if g.pathOps {
		class = " jsonb_path_ops"
	}
	return `CREATE INDEX IF NOT EXISTS "` + name + `" ON ` + tbl + ` USING GIN (` + col + class + `)`, nil
}

func jsonPathIndexSQL(table, column string, path []string, opts ...GINOption) (string, error) {
	g, tbl, _, err := indexParts(table, column, opts)
	if err != nil {
		return "", err
	}
	expr, err := JSONText(column, path...)
	if err != nil {
		return "", err
	}
	name := g.name
	if name == "" {
		name = "idx_" + table + "_" + column + "_" + strings.NewReplacer(".", "_", "-", "_", " ", "_").Replace(strings.Join(path, "_"))
	}
	return `CREATE INDEX IF NOT EXISTS "` + name + `" ON ` + tbl + ` ((` + expr + `))`, nil
}

func indexParts(table, column string, opts []GINOption) (ginIndex, string, string, error) {
	var g ginIndex
	for _, o := range opts {
		o(&g)
	}
	if !validIdentifier.MatchString(table) || strings.Contains(table, ".") {
		return g, "", "", errors.ErrInvalidColumnName.WithMeta(table)
	}
	if g.name != "" && (!validIdentifier.MatchString(g.name) || strings.Contains(g.name, ".")) {
		return g, "", "", errors.ErrInvalidColumnName.WithMeta(g.name)
	}
	col, err := quoteColumn(column)
	if err != nil {
		return g, "", "", err
	}
	if strings.Contains(column, ".") {
		return g, "", "", errors.ErrInvalidColumnName.WithMeta(column)
	}
	return g, `"` + table + `"`, col, nil
}

// quoteColumn checks a column name (optionally table.column) and quotes each
// part, so a mixed-case table name works.
func quoteColumn(column string) (string, error) {
	if !validIdentifier.MatchString(column) {
		return "", errors.ErrInvalidColumnName.WithMeta(column)
	}
	parts := strings.Split(column, ".")
	for i, p := range parts {
		parts[i] = `"` + p + `"`
	}
	return strings.Join(parts, "."), nil
}

// quotePathKeys quotes keys for a text[] literal: a key with a space or a dot
// must be in double quotes there.
func quotePathKeys(path []string) []string {
	out := make([]string, len(path))
	for i, k := range path {
		if strings.ContainsAny(k, " .") {
			out[i] = `"` + k + `"`
		} else {
			out[i] = k
		}
	}
	return out
}

// jsonText is value as ->> would return it.
func jsonText(value any) any {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	return strings.Trim(string(raw), `"`)
}
