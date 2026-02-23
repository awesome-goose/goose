package types

// Entity is a generic interface for working with typed entities
type Entity[T any] interface {
	// Hydrate initializes the entity with metadata
	Hydrate(
		name string,
		searchable []string,
		relations []string,
		scope func() (string, []any),
		unique func(*T) (any, []any),
		morphs map[string]func(*T),
		hooks map[string]func(*T) error,
		sort string,
	)

	// Name returns the table name for this entity
	Name() string

	// Exists checks if any entity matches the query
	Exists(query string, args ...any) (bool, error)

	// Count returns the number of entities matching the query
	Count(query string, args ...any) (int64, error)

	// Sum returns the sum of a column for entities matching the query
	Sum(column string, query string, args ...any) (int64, error)

	// Avg returns the average of a column for entities matching the query
	Avg(column string, query string, args ...any) (float64, error)

	// Min returns the minimum value of a column for entities matching the query
	Min(column string, query string, args ...any) (int64, error)

	// Max returns the maximum value of a column for entities matching the query
	Max(column string, query string, args ...any) (int64, error)

	// Insert creates multiple entities in the database
	Insert(entities ...*T) error

	// Upsert creates or updates entities based on primary key
	Upsert(entities ...*T) error

	// Update modifies entities matching the query
	Update(changes *T, query string, args ...any) (int64, error)

	// First retrieves the first entity matching the query
	First(query string, args ...any) (*T, error)

	// Last retrieves the last entity matching the query
	Last(query string, args ...any) (*T, error)

	// Some retrieves all entities matching the query
	Some(query string, args ...any) ([]T, error)

	// Page retrieves entities with pagination support
	Page(page int, perPage int, query string, args ...any) ([]T, error)

	// All retrieves all entities from the table
	All() ([]T, error)

	// Find retrieves entities with offset and limit
	Find(offset int, limit int, query string, args ...any) ([]T, error)

	// Delete removes entities matching the query
	Delete(query string, args ...any) (int64, error)

	// BuildQuery builds a query string from a map of query parameters
	BuildQuery(queries map[string]string) (string, []any)
}
