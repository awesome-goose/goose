package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/awesome-goose/goose/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseEntity struct {
	UUIDAware
	TimeAware
	SoftDeleteAware
}

func (e *BaseEntity) BeforeCreate(tx *gorm.DB) error {
	if e.Id == "" {
		e.Id = uuid.New().String()
	}
	now := time.Now().UTC()
	if e.CreatedAt == nil {
		e.CreatedAt = &now
	}
	e.UpdatedAt = &now
	return nil
}

func (e *BaseEntity) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now().UTC()
	e.UpdatedAt = &now
	return nil
}

// Entity is a generic repository for working with typed entities
type Entity[T any] struct {
	name       string
	searchable []string
	relations  []string
	scope      func() (string, []any)
	unique     func(*T) (any, []any)
	morphs     map[string]func(*T)
	hooks      map[string]func(*T) error
	sort       string

	log   types.Log `inject:""`
	query *Query    `inject:""`
}

func (r *Entity[T]) Hydrate(
	name string,
	searchable []string,
	relations []string,
	scope func() (string, []any),
	unique func(*T) (any, []any),
	morphs map[string]func(*T),
	hooks map[string]func(*T) error,
	sort string,
) {
	r.name = name
	r.searchable = searchable
	r.relations = relations
	r.scope = scope
	r.unique = unique
	r.morphs = morphs
	r.hooks = hooks
	r.sort = sort
}

// With returns a copy of Entity using the given database connection
func (r *Entity[T]) With(query *Query) *Entity[T] {
	clone := *r
	clone.query = query
	return &clone
}

// Name returns the table name for this entity (derived from struct name)
func (r *Entity[T]) Name() string {
	if r.name != "" {
		return r.name
	}

	var entity T
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check if entity implements Name() method
	if tabler, ok := any(&entity).(interface{ Name() string }); ok {
		r.name = tabler.Name()
		return r.name
	}

	// Convert struct name to snake_case and pluralize
	r.name = toName(t.Name())
	return r.name
}

// Tx executes a function within a database transaction
func (r *Entity[T]) Tx(fn func(*Entity[T]) error) error {
	return r.query.db.Transaction(func(tx *gorm.DB) error {
		txEntity := &Entity[T]{
			name:       r.name,
			searchable: r.searchable,
			relations:  r.relations,
			scope:      r.scope,
			unique:     r.unique,
			morphs:     r.morphs,
			hooks:      r.hooks,
			sort:       r.sort,
			query:      &Query{db: &Db{DB: tx, log: r.query.db.log}, log: r.log},
			log:        r.log,
		}
		return fn(txEntity)
	})
}

// Exists checks if any entity matches the query.
// If no args provided, treats query as an ID.
func (r *Entity[T]) Exists(query string, args ...any) (bool, error) {
	var count int64
	entity := new(T)

	q := r.query.db.Model(entity)

	// If no args, treat query as ID
	if len(args) == 0 {
		q = q.Where("id = ?", query)
	} else {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	err := q.Limit(1).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Count returns the number of entities matching the query.
// If no args provided, treats query as an ID.
func (r *Entity[T]) Count(query string, args ...any) (int64, error) {
	var count int64
	entity := new(T)

	q := r.query.db.Model(entity)

	// If no args, treat query as ID
	if len(args) == 0 && query != "" {
		q = q.Where("id = ?", query)
	} else if query != "" {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	err := q.Count(&count).Error
	return count, err
}

// Sum returns the sum of a column for entities matching the query
func (r *Entity[T]) Sum(column string, query string, args ...any) (int64, error) {
	if !validIdentifier.MatchString(column) {
		return 0, fmt.Errorf("invalid column name: %s", column)
	}

	entity := new(T)
	var result sql.NullInt64

	q := r.query.db.Model(entity)

	if query != "" {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	err := q.Select("COALESCE(SUM(" + column + "), 0)").Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.Int64, nil
}

// Avg returns the average of a column for entities matching the query
func (r *Entity[T]) Avg(column string, query string, args ...any) (float64, error) {
	if !validIdentifier.MatchString(column) {
		return 0, fmt.Errorf("invalid column name: %s", column)
	}

	entity := new(T)
	var result sql.NullFloat64

	q := r.query.db.Model(entity)

	if query != "" {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	err := q.Select("COALESCE(AVG(" + column + "), 0)").Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.Float64, nil
}

// Min returns the minimum value of a column for entities matching the query
func (r *Entity[T]) Min(column string, query string, args ...any) (int64, error) {
	if !validIdentifier.MatchString(column) {
		return 0, fmt.Errorf("invalid column name: %s", column)
	}

	entity := new(T)
	var result sql.NullInt64

	q := r.query.db.Model(entity)

	if query != "" {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	err := q.Select("MIN(" + column + ")").Scan(&result).Error
	if err != nil {
		return 0, err
	}
	if !result.Valid {
		return 0, ErrNotFound
	}
	return result.Int64, nil
}

// Max returns the maximum value of a column for entities matching the query
func (r *Entity[T]) Max(column string, query string, args ...any) (int64, error) {
	if !validIdentifier.MatchString(column) {
		return 0, fmt.Errorf("invalid column name: %s", column)
	}

	entity := new(T)
	var result sql.NullInt64

	q := r.query.db.Model(entity)

	if query != "" {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	err := q.Select("MAX(" + column + ")").Scan(&result).Error
	if err != nil {
		return 0, err
	}
	if !result.Valid {
		return 0, ErrNotFound
	}
	return result.Int64, nil
}

// Insert creates multiple entities in the database
func (r *Entity[T]) Insert(entities ...*T) error {
	if len(entities) == 0 {
		return nil
	}

	// Call BeforeCreate morphs and hooks
	for _, entity := range entities {
		if morph, ok := r.morphs[BeforeCreate]; ok {
			morph(entity)
		}
		if hook, ok := r.hooks[BeforeCreate]; ok {
			if err := hook(entity); err != nil {
				return err
			}
		}
	}

	if err := r.query.db.Create(&entities).Error; err != nil {
		return err
	}

	// Call AfterCreate morphs and hooks
	for _, entity := range entities {
		if morph, ok := r.morphs[AfterCreate]; ok {
			morph(entity)
		}
		if hook, ok := r.hooks[AfterCreate]; ok {
			if err := hook(entity); err != nil {
				return err
			}
		}
	}

	return nil
}

// Upsert creates or updates entities based on primary key
func (r *Entity[T]) Upsert(entities ...*T) error {
	if len(entities) == 0 {
		return nil
	}

	for _, entity := range entities {
		// Call BeforeCreate morph (used for both insert and update in upsert)
		if morph, ok := r.morphs[BeforeCreate]; ok {
			morph(entity)
		}

		// Call BeforeCreate hook (used for both insert and update in upsert)
		if hook, ok := r.hooks[BeforeCreate]; ok {
			if err := hook(entity); err != nil {
				return err
			}
		}

		if err := r.query.db.Save(entity).Error; err != nil {
			return err
		}

		// Call AfterCreate morph
		if morph, ok := r.morphs[AfterCreate]; ok {
			morph(entity)
		}

		// Call AfterCreate hook
		if hook, ok := r.hooks[AfterCreate]; ok {
			if err := hook(entity); err != nil {
				return err
			}
		}
	}

	return nil
}

// Update modifies entities matching the query.
// If no args provided, treats query as an ID.
func (r *Entity[T]) Update(changes *T, query string, args ...any) (int64, error) {
	// Call BeforeUpdate morph if registered
	if morph, ok := r.morphs[BeforeUpdate]; ok {
		morph(changes)
	}

	// Call BeforeUpdate hook if registered
	if hook, ok := r.hooks[BeforeUpdate]; ok {
		if err := hook(changes); err != nil {
			return 0, err
		}
	}

	q := r.query.db.Model(new(T))

	// If no args, treat query as ID
	if len(args) == 0 {
		q = q.Where("id = ?", query)
	} else {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	result := q.Updates(changes)
	if result.Error != nil {
		return 0, result.Error
	}

	// Call AfterUpdate morph if registered
	if morph, ok := r.morphs[AfterUpdate]; ok {
		morph(changes)
	}

	// Call AfterUpdate hook if registered
	if hook, ok := r.hooks[AfterUpdate]; ok {
		if err := hook(changes); err != nil {
			return result.RowsAffected, err
		}
	}

	return result.RowsAffected, nil
}

// First retrieves the first entity matching the query.
// If no args provided, treats query as an ID.
func (r *Entity[T]) First(query string, args ...any) (*T, error) {
	entity := new(T)

	// Call BeforeRead morph if registered
	if morph, ok := r.morphs[BeforeRead]; ok {
		morph(entity)
	}

	// Call BeforeRead hook if registered
	if hook, ok := r.hooks[BeforeRead]; ok {
		if err := hook(entity); err != nil {
			return nil, err
		}
	}

	q := r.query.db.Model(entity)

	// If no args, treat query as ID
	if len(args) == 0 {
		q = q.Where("id = ?", query)
	} else if query != "" {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	for _, relation := range r.relations {
		q = q.Preload(relation)
	}

	if r.sort != "" {
		q = q.Order(r.sort)
	}

	err := q.First(entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Call AfterRead morph if registered
	if morph, ok := r.morphs[AfterRead]; ok {
		morph(entity)
	}

	// Call AfterRead hook if registered
	if hook, ok := r.hooks[AfterRead]; ok {
		if err := hook(entity); err != nil {
			return nil, err
		}
	}

	return entity, nil
}

// Last retrieves the last entity matching the query.
// If no args provided, treats query as an ID.
func (r *Entity[T]) Last(query string, args ...any) (*T, error) {
	entity := new(T)

	// Call BeforeRead morph if registered
	if morph, ok := r.morphs[BeforeRead]; ok {
		morph(entity)
	}

	// Call BeforeRead hook if registered
	if hook, ok := r.hooks[BeforeRead]; ok {
		if err := hook(entity); err != nil {
			return nil, err
		}
	}

	q := r.query.db.Model(entity)

	// If no args, treat query as ID
	if len(args) == 0 {
		q = q.Where("id = ?", query)
	} else if query != "" {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	for _, relation := range r.relations {
		q = q.Preload(relation)
	}

	if r.sort != "" {
		q = q.Order(r.sort)
	}

	err := q.Last(entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Call AfterRead morph if registered
	if morph, ok := r.morphs[AfterRead]; ok {
		morph(entity)
	}

	// Call AfterRead hook if registered
	if hook, ok := r.hooks[AfterRead]; ok {
		if err := hook(entity); err != nil {
			return nil, err
		}
	}

	return entity, nil
}

// Some retrieves all entities matching the query
func (r *Entity[T]) Some(query string, args ...any) ([]T, error) {
	return r.Find(-1, -1, query, args...)
}

// Page retrieves entities with pagination support
func (r *Entity[T]) Page(page int, perPage int, query string, args ...any) ([]T, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	offset := (page - 1) * perPage
	limit := perPage

	return r.Find(offset, limit, query, args...)
}

// All retrieves all entities from the table
func (r *Entity[T]) All() ([]T, error) {
	return r.Find(-1, -1, "")
}

// Find retrieves entities with pagination support
func (r *Entity[T]) Find(offset int, limit int, query string, args ...any) ([]T, error) {
	var entities []T
	entity := new(T)

	// Call BeforeRead morph if registered
	if morph, ok := r.morphs[BeforeRead]; ok {
		morph(entity)
	}

	// Call BeforeRead hook if registered
	if hook, ok := r.hooks[BeforeRead]; ok {
		if err := hook(entity); err != nil {
			return nil, err
		}
	}

	q := r.query.db.Model(entity)

	if query != "" {
		q = q.Where(query, args...)
	}

	// Apply scope if defined
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	for _, relation := range r.relations {
		q = q.Preload(relation)
	}

	if r.sort != "" {
		q = q.Order(r.sort)
	}

	if limit > 0 {
		q = q.Limit(limit)
	}

	if offset > 0 {
		q = q.Offset(offset)
	}

	err := q.Find(&entities).Error
	if err != nil {
		return nil, err
	}

	// Call AfterRead morph for each entity
	if morph, ok := r.morphs[AfterRead]; ok {
		for i := range entities {
			morph(&entities[i])
		}
	}

	// Call AfterRead hook for each entity
	if hook, ok := r.hooks[AfterRead]; ok {
		for i := range entities {
			if err := hook(&entities[i]); err != nil {
				return nil, err
			}
		}
	}

	return entities, nil
}

// Delete removes entities matching the query.
// If no args provided, treats query as an ID.
func (r *Entity[T]) Delete(query string, args ...any) (int64, error) {
	entity := new(T)
	var q *gorm.DB

	// If no args, treat query as ID and fetch entity first to support hooks
	if len(args) == 0 {
		// Use a separate query to fetch the entity
		fetchQuery := r.query.db.Model(entity).Where("id = ?", query)

		// Apply scope to fetch query as well
		if r.scope != nil {
			scopeQuery, scopeArgs := r.scope()
			if scopeQuery != "" {
				fetchQuery = fetchQuery.Where(scopeQuery, scopeArgs...)
			}
		}

		if err := fetchQuery.First(entity).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, nil
			}
			return 0, err
		}

		// Reconstruct query for delete to ensure consistency
		q = r.query.db.Model(entity).Where("id = ?", query)
	} else {
		q = r.query.db.Model(entity).Where(query, args...)
	}

	// Apply scope if defined (applied to both paths)
	if r.scope != nil {
		scopeQuery, scopeArgs := r.scope()
		if scopeQuery != "" {
			q = q.Where(scopeQuery, scopeArgs...)
		}
	}

	// Call BeforeDelete morph if registered
	if morph, ok := r.morphs[BeforeDelete]; ok {
		morph(entity)
	}

	// Call BeforeDelete hook if registered
	if hook, ok := r.hooks[BeforeDelete]; ok {
		if err := hook(entity); err != nil {
			return 0, err
		}
	}

	result := q.Delete(entity)
	if result.Error != nil {
		return 0, result.Error
	}

	// Call AfterDelete morph if registered
	if morph, ok := r.morphs[AfterDelete]; ok {
		morph(entity)
	}

	// Call AfterDelete hook if registered
	if hook, ok := r.hooks[AfterDelete]; ok {
		if err := hook(entity); err != nil {
			return result.RowsAffected, err
		}
	}

	return result.RowsAffected, nil
}

func (r *Entity[T]) BuildQuery(queries map[string]string) (string, []any) {
	var (
		scopeQuery string
		scopeArgs  []any
	)
	if r.scope != nil {
		scopeQuery, scopeArgs = r.scope()
	}

	return BuildQuery(queries, r.searchable, r.Name(), scopeQuery, scopeArgs)
}
