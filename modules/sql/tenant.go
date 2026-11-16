package sql

import (
	"context"
	"fmt"
	"reflect"

	"github.com/awesome-goose/goose/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DefaultTenantColumn is the column ForTenant binds when RequireTenant named
// none.
const DefaultTenantColumn = "workspace_id"

// tenantState is an Entity's tenant binding. The zero value is an entity with
// no tenant rules: it behaves exactly as one did before tenant scoping existed.
type tenantState struct {
	// column is the tenant column ("" until RequireTenant or Scoped names one).
	column string
	// required: every operation fails with ErrTenantRequired until the entity
	// is bound (ForTenant, Scoped) or opened (AcrossTenants).
	required bool
	// bound: reads, updates and deletes are confined to value, and writes are
	// stamped with it.
	bound bool
	value any
	// open: AcrossTenants was called; the guard is off for this copy.
	open bool
	// baseScope is the scope the entity had before it was bound, kept so that
	// re-binding or opening a bound copy can restore it.
	baseScope func() (string, []any)
}

func (t tenantState) base() func() (string, []any) { return t.baseScope }

// RequireTenant marks the entity tenant-owned, in place: from now on every
// operation on it fails with errors.ErrTenantRequired unless it goes through
// ForTenant (or Scoped), or through AcrossTenants for a job that really spans
// tenants. Call it once, from the entity's OnRegister, next to Hydrate:
//
//	func (e *ContactEntity) OnRegister() {
//		e.Hydrate("Contacts", ...)
//		e.RequireTenant("workspace_id")
//	}
//
// and then, per request:
//
//	rows, err := s.contacts.ForTenant(ctx.WorkspaceID).Find(0, 50, "")
//
// So a query that forgot its tenant fails loudly instead of reading every
// tenant's rows. column defaults to DefaultTenantColumn when empty.
func (r *Entity[T]) RequireTenant(column string) {
	if column == "" {
		column = DefaultTenantColumn
	}
	r.tenant.column = column
	r.tenant.required = true
}

// ForTenant returns a copy of the entity bound to one tenant, on the column
// RequireTenant named (DefaultTenantColumn otherwise). See Scoped.
func (r *Entity[T]) ForTenant(id any) *Entity[T] {
	column := r.tenant.column
	if column == "" {
		column = DefaultTenantColumn
	}
	return r.Scoped(column, id)
}

// Scoped returns a copy of the entity bound to column = value:
//
//   - every read, count, aggregate, update and delete carries
//     "AND column = value", on top of the scope given to Hydrate;
//   - Insert and Upsert set the column on rows that leave it empty, and refuse
//     (errors.ErrTenantMismatch) a row that names another value;
//   - Update refuses changes that would move a row to another value;
//   - Upsert refuses to overwrite a row that exists under another value.
//
// It fails closed: an empty value (nil, "", 0) binds nothing, and every
// operation on the copy returns errors.ErrTenantRequired. The receiver is not
// changed, so one injected entity serves every request.
func (r *Entity[T]) Scoped(column string, value any) *Entity[T] {
	clone := *r
	clone.tenant.column = column
	clone.tenant.open = false
	if !validIdentifier.MatchString(column) || isZero(value) {
		// Keep the guard on: tenantGuard rejects every operation.
		clone.tenant.required = true
		clone.tenant.bound = false
		clone.tenant.value = nil
		return &clone
	}
	clone.tenant.bound = true
	clone.tenant.value = value

	base := r.scope
	if r.tenant.bound {
		// Re-binding (ForTenant on an already bound copy) replaces the
		// binding; it must not AND two tenants together or drop the base scope.
		base = r.tenant.base()
	}
	clone.tenant.baseScope = base
	clone.scope = func() (string, []any) {
		clause := column + " = ?"
		if base == nil {
			return clause, []any{value}
		}
		q, args := base()
		if q == "" {
			return clause, []any{value}
		}
		return "(" + q + ") AND " + clause, append(append([]any{}, args...), value)
	}
	return &clone
}

// AcrossTenants returns a copy with the tenant guard off, for the few jobs
// that must span tenants: a retention sweep, a migration, a system report.
// Naming it makes every such place easy to find and review.
func (r *Entity[T]) AcrossTenants() *Entity[T] {
	clone := *r
	if r.tenant.bound {
		clone.scope = r.tenant.base()
	}
	clone.tenant.bound = false
	clone.tenant.value = nil
	clone.tenant.baseScope = nil
	clone.tenant.open = true
	return &clone
}

// RequiresTenant reports whether RequireTenant was called, and its column:
// a generic layer (core/crud) uses it to bind the entity to the caller's
// tenant before every query.
func (r *Entity[T]) RequiresTenant() (column string, required bool) {
	return r.tenant.column, r.tenant.required
}

// TenantBound reports the tenant value the entity is bound to, if any.
func (r *Entity[T]) TenantBound() (any, bool) {
	return r.tenant.value, r.tenant.bound
}

// tenantGuard is the check every Entity operation starts with.
func (r *Entity[T]) tenantGuard() error {
	if r.tenant.required && !r.tenant.bound && !r.tenant.open {
		return errors.ErrTenantRequired.WithMeta(r.Name())
	}
	return nil
}

// tenantField is T's field for the bound column, or nil when T has none (a
// row scoped through a parent has no tenant column to stamp).
func (r *Entity[T]) tenantField() *schema.Field {
	if !r.tenant.bound || r.query == nil || r.query.db == nil || r.query.db.DB == nil {
		return nil
	}
	stmt := &gorm.Statement{DB: r.query.db.DB}
	if err := stmt.Parse(new(T)); err != nil || stmt.Schema == nil {
		return nil
	}
	return stmt.Schema.LookUpField(r.tenant.column)
}

// tenantCheck refuses a row that names a tenant other than the bound one. A
// row that names none passes.
func (r *Entity[T]) tenantCheck(entity *T) error {
	f := r.tenantField()
	if f == nil || entity == nil {
		return nil
	}
	got, zero := f.ValueOf(context.Background(), reflect.ValueOf(entity).Elem())
	if zero || sameValue(got, r.tenant.value) {
		return nil
	}
	return errors.ErrTenantMismatch.WithMeta(r.Name())
}

// tenantStamp is tenantCheck, and it sets the tenant on a row that names none.
func (r *Entity[T]) tenantStamp(entity *T) error {
	f := r.tenantField()
	if f == nil || entity == nil {
		return nil
	}
	rv := reflect.ValueOf(entity).Elem()
	got, zero := f.ValueOf(context.Background(), rv)
	if zero {
		if err := f.Set(context.Background(), rv, r.tenant.value); err != nil {
			return errors.ErrTenantMismatch.WithError(err).WithMeta(r.Name())
		}
		return nil
	}
	if sameValue(got, r.tenant.value) {
		return nil
	}
	return errors.ErrTenantMismatch.WithMeta(r.Name())
}

// tenantOwns refuses an Upsert that would overwrite a row stored under
// another tenant: Upsert saves by primary key, which the scope does not
// filter. Soft-deleted rows count: saving over one would revive it here.
func (r *Entity[T]) tenantOwns(entity *T) error {
	f := r.tenantField()
	if f == nil || entity == nil {
		return nil
	}
	stmt := &gorm.Statement{DB: r.query.db.DB}
	if err := stmt.Parse(new(T)); err != nil || stmt.Schema == nil {
		return nil
	}
	pk := stmt.Schema.PrioritizedPrimaryField
	if pk == nil {
		return nil
	}
	id, zero := pk.ValueOf(context.Background(), reflect.ValueOf(entity).Elem())
	if zero {
		return nil
	}
	var n int64
	err := r.query.db.Model(new(T)).Unscoped().
		Where(pk.DBName+" = ?", id).
		Where("("+r.tenant.column+" <> ? OR "+r.tenant.column+" IS NULL)", r.tenant.value).
		Limit(1).Count(&n).Error
	if err != nil {
		return err
	}
	if n > 0 {
		return errors.ErrTenantMismatch.WithMeta(r.Name())
	}
	return nil
}

func sameValue(a, b any) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func isZero(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		return rv.IsNil() || isZero(rv.Elem().Interface())
	}
	return rv.IsZero()
}
