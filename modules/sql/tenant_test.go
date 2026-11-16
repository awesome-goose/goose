package sql

import (
	stderrors "errors"
	"testing"

	"github.com/awesome-goose/goose/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type tenantRow struct {
	BaseEntity
	WorkspaceID string `gorm:"column:workspace_id;index" json:"workspaceId"`
	Name        string `gorm:"column:name" json:"name"`
	Amount      int64  `gorm:"column:amount" json:"amount"`
}

func tenantEntity(t *testing.T) *Entity[tenantRow] {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&tenantRow{}); err != nil {
		t.Fatal(err)
	}
	e := NewEntity[tenantRow](NewQuery(db))
	e.Hydrate("tenant_rows", nil, nil, nil, nil, nil, nil, "")
	return e
}

func isErr(err error, want *errors.Error) bool {
	var e *errors.Error
	return stderrors.As(err, &e) && e.Code == want.Code
}

// The cross-tenant test: two tenants with rows of the same shape. Whatever one
// tenant's bound entity does, it never reads, counts, changes or removes the
// other's rows.
func TestScopedEntityNeverCrossesTenants(t *testing.T) {
	e := tenantEntity(t)
	a, b := e.ForTenant("ws-a"), e.ForTenant("ws-b")

	rowA := &tenantRow{Name: "a1", Amount: 10}
	if err := a.Insert(rowA, &tenantRow{Name: "a2", Amount: 5}); err != nil {
		t.Fatal(err)
	}
	rowB := &tenantRow{Name: "b1", Amount: 100}
	if err := b.Insert(rowB); err != nil {
		t.Fatal(err)
	}
	if rowA.WorkspaceID != "ws-a" || rowB.WorkspaceID != "ws-b" {
		t.Fatalf("insert did not stamp the tenant: %q %q", rowA.WorkspaceID, rowB.WorkspaceID)
	}

	// reads
	if rows, err := a.All(); err != nil || len(rows) != 2 {
		t.Fatalf("a.All = %d rows, %v; want 2", len(rows), err)
	}
	if got, err := a.First(rowB.Id); err == nil || got != nil {
		t.Fatalf("a read b's row by id: %+v", got)
	}
	if got, err := a.First("name = ?", "b1"); err == nil || got != nil {
		t.Fatalf("a read b's row by query: %+v", got)
	}
	if got, _ := a.Last("name = ?", "b1"); got != nil {
		t.Fatalf("a.Last read b's row: %+v", got)
	}
	if ok, _ := a.Exists(rowB.Id); ok {
		t.Fatal("a.Exists saw b's row")
	}
	if n, _ := a.Count(""); n != 2 {
		t.Fatalf("a.Count = %d, want 2", n)
	}
	if n, _ := a.Count("name LIKE ?", "%1"); n != 1 {
		t.Fatalf("a.Count(name like %%1) = %d, want 1 (b1 is not a's)", n)
	}
	if s, _ := a.Sum("amount", ""); s != 15 {
		t.Fatalf("a.Sum = %d, want 15", s)
	}
	if m, _ := a.Max("amount", ""); m != 10 {
		t.Fatalf("a.Max = %d, want 10", m)
	}
	// an OR in the caller's query cannot widen the scope
	if rows, _ := a.Find(0, 10, "name = ? OR 1 = 1", "zzz"); len(rows) != 2 {
		t.Fatalf("an OR widened the scope: %d rows", len(rows))
	}

	// writes
	if n, err := a.Update(&tenantRow{Name: "hacked"}, rowB.Id); err != nil || n != 0 {
		t.Fatalf("a updated b's row: n=%d err=%v", n, err)
	}
	if n, err := a.Update(&tenantRow{Name: "hacked"}, "name = ?", "b1"); err != nil || n != 0 {
		t.Fatalf("a updated b's row by query: n=%d err=%v", n, err)
	}
	if _, err := a.Update(&tenantRow{WorkspaceID: "ws-b"}, rowA.Id); !isErr(err, errors.ErrTenantMismatch) {
		t.Fatalf("a moved its row to b: %v", err)
	}
	if n, err := a.Delete(rowB.Id); err != nil || n != 0 {
		t.Fatalf("a deleted b's row: n=%d err=%v", n, err)
	}
	if n, err := a.Delete("name = ?", "b1"); err != nil || n != 0 {
		t.Fatalf("a deleted b's row by query: n=%d err=%v", n, err)
	}
	if err := a.Insert(&tenantRow{WorkspaceID: "ws-b", Name: "planted"}); !isErr(err, errors.ErrTenantMismatch) {
		t.Fatalf("a inserted a row for b: %v", err)
	}
	// Upsert saves by primary key: it must not overwrite b's row.
	if err := a.Upsert(&tenantRow{BaseEntity: BaseEntity{UUIDAware: UUIDAware{Id: rowB.Id}}, Name: "taken"}); !isErr(err, errors.ErrTenantMismatch) {
		t.Fatalf("a upserted over b's row: %v", err)
	}
	rowA.Name = "a1-renamed"
	if err := a.Upsert(rowA); err != nil {
		t.Fatalf("a could not upsert its own row: %v", err)
	}

	// b is untouched
	got, err := b.First(rowB.Id)
	if err != nil || got.Name != "b1" || got.WorkspaceID != "ws-b" {
		t.Fatalf("b's row changed: %+v %v", got, err)
	}
	if n, _ := b.Count(""); n != 1 {
		t.Fatalf("b.Count = %d, want 1", n)
	}
	// and a transaction keeps the binding
	err = a.Tx(func(tx *Entity[tenantRow]) error {
		if n, _ := tx.Count(""); n != 2 {
			t.Fatalf("tx.Count = %d, want 2", n)
		}
		return tx.Insert(&tenantRow{Name: "a3"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := e.AcrossTenants().First("name = ?", "a3"); got == nil || got.WorkspaceID != "ws-a" {
		t.Fatalf("tx insert was not stamped: %+v", got)
	}
}

func TestRequireTenantRejectsEveryUnscopedOperation(t *testing.T) {
	e := tenantEntity(t)
	if err := e.ForTenant("ws-a").Insert(&tenantRow{Name: "a1"}); err != nil {
		t.Fatal(err)
	}
	e.RequireTenant("") // workspace_id

	checks := map[string]func() error{
		"Exists": func() error { _, err := e.Exists("x"); return err },
		"Count":  func() error { _, err := e.Count(""); return err },
		"Sum":    func() error { _, err := e.Sum("amount", ""); return err },
		"Avg":    func() error { _, err := e.Avg("amount", ""); return err },
		"Min":    func() error { _, err := e.Min("amount", ""); return err },
		"Max":    func() error { _, err := e.Max("amount", ""); return err },
		"First":  func() error { _, err := e.First("x"); return err },
		"Last":   func() error { _, err := e.Last("x"); return err },
		"Find":   func() error { _, err := e.Find(0, 10, ""); return err },
		"Some":   func() error { _, err := e.Some(""); return err },
		"Page":   func() error { _, err := e.Page(1, 10, ""); return err },
		"All":    func() error { _, err := e.All(); return err },
		"Insert": func() error { return e.Insert(&tenantRow{Name: "x"}) },
		"Upsert": func() error { return e.Upsert(&tenantRow{Name: "x"}) },
		"Update": func() error { _, err := e.Update(&tenantRow{Name: "x"}, "name = ?", "a1"); return err },
		"Delete": func() error { _, err := e.Delete("name = ?", "a1"); return err },
		"Tx": func() error {
			return e.Tx(func(tx *Entity[tenantRow]) error { _, err := tx.All(); return err })
		},
		// an empty tenant binds nothing: it must not mean "all tenants"
		"ForTenant(\"\")": func() error { _, err := e.ForTenant("").All(); return err },
		"ForTenant(nil)":  func() error { _, err := e.ForTenant(nil).All(); return err },
	}
	for name, run := range checks {
		if err := run(); !isErr(err, errors.ErrTenantRequired) {
			t.Errorf("%s on an unbound tenant-owned entity: err = %v, want ErrTenantRequired", name, err)
		}
	}
	if q, _ := e.BuildQuery(map[string]string{}); q != "1 = 0" && q != "(1 = 0)" {
		t.Errorf("BuildQuery on an unbound entity = %q, want a clause that matches nothing", q)
	}

	// bound and explicitly open copies work; the injected entity is unchanged
	if rows, err := e.ForTenant("ws-a").All(); err != nil || len(rows) != 1 {
		t.Fatalf("bound: %d rows, %v", len(rows), err)
	}
	if rows, err := e.AcrossTenants().All(); err != nil || len(rows) != 1 {
		t.Fatalf("AcrossTenants: %d rows, %v", len(rows), err)
	}
	if _, err := e.All(); !isErr(err, errors.ErrTenantRequired) {
		t.Fatal("binding a copy changed the injected entity")
	}
}

// An entity without tenant rules behaves as before; Scoped keeps a Hydrate
// scope, and re-binding replaces the tenant rather than adding a second one.
func TestScopedComposesWithTheHydrateScope(t *testing.T) {
	e := tenantEntity(t)
	e.Hydrate("tenant_rows", nil, nil, func() (string, []any) { return "amount > ?", []any{1} }, nil, nil, nil, "")
	if err := e.Insert(&tenantRow{WorkspaceID: "ws-a", Name: "big", Amount: 5},
		&tenantRow{WorkspaceID: "ws-a", Name: "small", Amount: 1},
		&tenantRow{WorkspaceID: "ws-b", Name: "bigb", Amount: 7}); err != nil {
		t.Fatal(err)
	}
	if n, _ := e.Count(""); n != 2 {
		t.Fatalf("unbound Count = %d, want 2 (the Hydrate scope only)", n)
	}
	a := e.ForTenant("ws-a")
	if n, _ := a.Count(""); n != 1 {
		t.Fatalf("a.Count = %d, want 1 (scope AND tenant)", n)
	}
	if n, _ := a.ForTenant("ws-b").Count(""); n != 1 {
		t.Fatalf("re-bound Count = %d, want 1", n)
	}
	if n, _ := a.AcrossTenants().Count(""); n != 2 {
		t.Fatalf("AcrossTenants from a bound copy = %d, want 2", n)
	}
	if v, ok := a.TenantBound(); !ok || v != "ws-a" {
		t.Fatalf("TenantBound = %v %v", v, ok)
	}
}
