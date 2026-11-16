package sql

import (
	"context"
	stderrors "errors"
	"testing"
)

// A query run under a cancelled request context does not reach the
// database.
func TestWithContextStopsACancelledQuery(t *testing.T) {
	e := tenantEntity(t).ForTenant("ws-a")
	if err := e.Insert(&tenantRow{Name: "a1"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if rows, err := e.WithContext(ctx).All(); err != nil || len(rows) != 1 {
		t.Fatalf("live context: %d rows, %v", len(rows), err)
	}
	cancel()
	if _, err := e.WithContext(ctx).All(); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context: err = %v, want context.Canceled", err)
	}
	if _, err := e.WithContext(ctx).Count(""); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context (Count): err = %v", err)
	}
	q := NewQuery(e.query.db.DB).WithContext(ctx)
	if _, err := q.Count("tenant_rows", "1 = 1"); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Query: err = %v", err)
	}
	// the tenant binding survives WithContext
	if v, ok := e.WithContext(context.Background()).TenantBound(); !ok || v != "ws-a" {
		t.Fatal("WithContext dropped the tenant binding")
	}
}
