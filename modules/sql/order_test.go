package sql

import "testing"

func TestOrderBy(t *testing.T) {
	e := tenantEntity(t).ForTenant("ws-a")
	for _, n := range []string{"b", "c", "a"} {
		if err := e.Insert(&tenantRow{Name: n}); err != nil {
			t.Fatal(err)
		}
	}
	asc, err := e.OrderBy("name")
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := asc.All()
	if len(rows) != 3 || rows[0].Name != "a" || rows[2].Name != "c" {
		t.Fatalf("asc: %+v", rows)
	}
	desc, _ := e.OrderBy("name desc, id ASC")
	if first, _ := desc.First("1 = ?", 1); first == nil || first.Name != "c" {
		t.Fatalf("desc first: %+v", first)
	}
	if v, ok := desc.TenantBound(); !ok || v != "ws-a" {
		t.Fatal("OrderBy dropped the tenant binding")
	}
	for _, bad := range []string{"", "name; DROP TABLE x", "name DESC NULLS", "name sideways", "(select 1)", "name,"} {
		if _, err := e.OrderBy(bad); err == nil {
			t.Errorf("OrderBy(%q) accepted", bad)
		}
	}
}

func TestRequiresTenant(t *testing.T) {
	e := tenantEntity(t)
	if _, req := e.RequiresTenant(); req {
		t.Fatal("a plain entity requires a tenant")
	}
	e.RequireTenant("org_id")
	if col, req := e.RequiresTenant(); !req || col != "org_id" {
		t.Fatalf("RequiresTenant = %q %v", col, req)
	}
}
