package sql

import (
	"os"
	"strconv"
	"testing"

	"github.com/awesome-goose/goose/errors"
	"gorm.io/gorm"
)

func TestJSONHelpersBuildClauses(t *testing.T) {
	if got, err := JSONText("meta", "priority"); err != nil || got != `"meta"->>'priority'` {
		t.Fatalf("JSONText = %q %v", got, err)
	}
	if got, err := JSONText("Tasks.meta", "a", "b c"); err != nil || got != `"Tasks"."meta" #>> '{a,"b c"}'` {
		t.Fatalf("JSONText nested = %q %v", got, err)
	}
	q, args, err := JSONEq("meta", "count", 42)
	if err != nil || q != `"meta"->>'count' = ?` || len(args) != 1 || args[0] != "42" {
		t.Fatalf("JSONEq = %q %v %v", q, args, err)
	}
	q, args, err = JSONContains("labels", []string{"urgent"})
	if err != nil || q != `"labels" @> ?::jsonb` || args[0] != `["urgent"]` {
		t.Fatalf("JSONContains = %q %v %v", q, args, err)
	}
	q, args, err = JSONHasKey("meta", "owner")
	if err != nil || q != `jsonb_exists("meta", ?)` || args[0] != "owner" {
		t.Fatalf("JSONHasKey = %q %v %v", q, args, err)
	}
	q, args, _ = JSONHasAnyKey("meta", "a", "b")
	if q != `jsonb_exists_any("meta", ARRAY[?,?]::text[])` || len(args) != 2 {
		t.Fatalf("JSONHasAnyKey = %q %v", q, args)
	}
	if q, _, _ := JSONHasAnyKey("meta"); q != "1 = 0" {
		t.Fatalf("JSONHasAnyKey with no keys = %q, want a clause that matches nothing", q)
	}
	if s, err := ginIndexSQL("Tasks", "meta"); err != nil || s != `CREATE INDEX IF NOT EXISTS "idx_Tasks_meta_gin" ON "Tasks" USING GIN ("meta")` {
		t.Fatalf("ginIndexSQL = %q %v", s, err)
	}
	if s, err := ginIndexSQL("Tasks", "meta", GINPathOps(), GINName("tasks_meta")); err != nil || s != `CREATE INDEX IF NOT EXISTS "tasks_meta" ON "Tasks" USING GIN ("meta" jsonb_path_ops)` {
		t.Fatalf("ginIndexSQL opts = %q %v", s, err)
	}
	if s, err := jsonPathIndexSQL("Tasks", "meta", []string{"priority"}); err != nil || s != `CREATE INDEX IF NOT EXISTS "idx_Tasks_meta_priority" ON "Tasks" (("meta"->>'priority'))` {
		t.Fatalf("jsonPathIndexSQL = %q %v", s, err)
	}
}

// Names and keys are written into the SQL, so anything that could end a
// literal or an identifier is refused; values are only ever bound.
func TestJSONHelpersRefuseInjection(t *testing.T) {
	bad := func(name string, err error, want *errors.Error) {
		t.Helper()
		if !isErr(err, want) {
			t.Errorf("%s: err = %v, want %s", name, err, want.Code)
		}
	}
	_, err := JSONText(`meta"; DROP TABLE x; --`, "a")
	bad("column", err, errors.ErrInvalidColumnName)
	_, err = JSONText("meta", `a' OR '1'='1`)
	bad("key with a quote", err, errors.ErrInvalidJSONPath)
	_, err = JSONText("meta", "a", "b}'::text; --")
	bad("key with a brace", err, errors.ErrInvalidJSONPath)
	_, err = JSONText("meta", "")
	bad("empty key", err, errors.ErrInvalidJSONPath)
	_, err = JSONText("meta")
	bad("no path", err, errors.ErrInvalidJSONPath)
	_, err = JSONText("meta", `a\`)
	bad("key with a backslash", err, errors.ErrInvalidJSONPath)
	_, _, err = JSONContains("meta) OR (1=1", 1)
	bad("contains column", err, errors.ErrInvalidColumnName)
	_, err = ginIndexSQL(`T" (x); --`, "meta")
	bad("table", err, errors.ErrInvalidColumnName)
	_, err = ginIndexSQL("Tasks", "meta", GINName(`i" ON x`))
	bad("index name", err, errors.ErrInvalidColumnName)
	bad("drop", DropIndex(&Query{}, `i"; DROP TABLE x`), errors.ErrInvalidColumnName)

	// a hostile value is an argument, never part of the clause
	q, args, err := JSONEq("meta", "k", `x' OR '1'='1`)
	if err != nil || q != `"meta"->>'k' = ?` || args[0] != `x' OR '1'='1` {
		t.Fatalf("value was not bound: %q %v %v", q, args, err)
	}
}

type jsonRow struct {
	BaseEntity
	WorkspaceID string `gorm:"column:workspace_id" json:"workspaceId"`
	Name        string `gorm:"column:name" json:"name"`
	Meta        string `gorm:"column:meta;type:jsonb" json:"meta"`
}

func (jsonRow) TableName() string { return "GooseJsonbRows" }

// Against a real Postgres (GOOSE_TEST_DB_HOST, optionally _PORT/_USER/_PASS/_NAME):
// the clauses select the right rows, compose with the tenant scope, and the
// index DDL runs twice without error.
func TestJSONHelpersAgainstPostgres(t *testing.T) {
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
	c := &Config{Dialect: "postgres", Host: host, Port: port, User: user, Pass: os.Getenv("GOOSE_TEST_DB_PASS"), Name: name, SSLMode: "disable"}
	db, err := gorm.Open(postgresDialector(PostgresDSN(c), c.TimeZone), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	q := NewQuery(db)
	_, _ = q.Exec(`DROP TABLE IF EXISTS "GooseJsonbRows"`)
	if _, err := q.Exec(`CREATE TABLE "GooseJsonbRows" (id varchar(36) PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, deleted_at timestamptz, workspace_id varchar(36), name text, meta jsonb)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = q.Exec(`DROP TABLE IF EXISTS "GooseJsonbRows"`) })

	for i := 0; i < 2; i++ { // idempotent
		if err := CreateGINIndex(q, "GooseJsonbRows", "meta"); err != nil {
			t.Fatal(err)
		}
		if err := CreateJSONPathIndex(q, "GooseJsonbRows", "meta", []string{"priority"}); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := q.Count("pg_indexes", "tablename = ? AND indexname IN ?", "GooseJsonbRows", []string{"idx_GooseJsonbRows_meta_gin", "idx_GooseJsonbRows_meta_priority"}); err != nil || n != 2 {
		t.Fatalf("indexes = %d %v, want 2", n, err)
	}

	e := NewEntity[jsonRow](q)
	e.Hydrate("GooseJsonbRows", nil, nil, nil, nil, nil, nil, "")
	e.RequireTenant("workspace_id")
	a, b := e.ForTenant("ws-a"), e.ForTenant("ws-b")
	if err := a.Insert(
		&jsonRow{Name: "a-high", Meta: `{"priority":"high","count":42,"labels":["urgent","x"],"owner":{"team":"ops","lead":"ada"}}`},
		&jsonRow{Name: "a-low", Meta: `{"priority":"low","labels":["later"]}`},
	); err != nil {
		t.Fatal(err)
	}
	if err := b.Insert(&jsonRow{Name: "b-high", Meta: `{"priority":"high","labels":["urgent"],"owner":{"team":"ops"}}`}); err != nil {
		t.Fatal(err)
	}

	names := func(e *Entity[jsonRow], where string, args []any, err error) []string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		rows, err := e.Some(where, args...)
		if err != nil {
			t.Fatalf("%s: %v", where, err)
		}
		var out []string
		for _, r := range rows {
			out = append(out, r.Name)
		}
		return out
	}
	one := func(got []string, want string) {
		t.Helper()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("got %v, want [%s]", got, want)
		}
	}
	w, args, err := JSONEq("meta", "priority", "high")
	one(names(a, w, args, err), "a-high") // not b-high: the tenant scope holds
	one(names(b, w, args, err), "b-high")
	w, args, err = JSONEq("meta", "count", 42)
	one(names(a, w, args, err), "a-high")
	w, args, err = JSONPathEq("meta", []string{"owner", "lead"}, "ada")
	one(names(a, w, args, err), "a-high")
	w, args, err = JSONContains("meta", map[string]any{"labels": []string{"urgent"}})
	one(names(a, w, args, err), "a-high")
	w, args, err = JSONContains("meta", map[string]any{"owner": map[string]any{"team": "ops"}})
	one(names(b, w, args, err), "b-high")
	w, args, err = JSONHasKey("meta", "owner")
	one(names(a, w, args, err), "a-high")
	w, args, err = JSONHasAnyKey("meta", "count", "nope")
	one(names(a, w, args, err), "a-high")
	// a hostile value matches nothing and breaks nothing
	w, args, err = JSONEq("meta", "priority", `high' OR '1'='1`)
	if got := names(a, w, args, err); len(got) != 0 {
		t.Fatalf("hostile value matched %v", got)
	}

	if err := DropIndex(q, "idx_GooseJsonbRows_meta_gin"); err != nil {
		t.Fatal(err)
	}
}
