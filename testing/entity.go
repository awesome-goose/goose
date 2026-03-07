package testing

import (
	"testing"

	"github.com/awesome-goose/goose/types"
)

// EntityTest provides utilities for testing entities
type EntityTest[E any] struct {
	t       *testing.T
	tHelper *T
	entity  types.Entity[E]
}

// NewEntityTest creates a new entity test helper
func NewEntityTest[E any](t *testing.T, entity types.Entity[E]) *EntityTest[E] {
	return &EntityTest[E]{
		t:       t,
		tHelper: New(t),
		entity:  entity,
	}
}

// Entity returns the underlying entity
func (et *EntityTest[E]) Entity() types.Entity[E] {
	return et.entity
}

// T returns the test helper for assertions
func (et *EntityTest[E]) T() *T {
	return et.tHelper
}

// TestExists tests if an entity exists matching the query
func (et *EntityTest[E]) TestExists(query string, args ...any) bool {
	et.t.Helper()
	exists, err := et.entity.Exists(query, args...)
	et.tHelper.Expect(err).ToBeNil()
	return exists
}

// TestCount tests the count of entities matching the query
func (et *EntityTest[E]) TestCount(query string, args ...any) int64 {
	et.t.Helper()
	count, err := et.entity.Count(query, args...)
	et.tHelper.Expect(err).ToBeNil()
	return count
}

// TestFirst tests retrieving the first entity matching the query
func (et *EntityTest[E]) TestFirst(query string, args ...any) *E {
	et.t.Helper()
	entity, err := et.entity.First(query, args...)
	et.tHelper.Expect(err).ToBeNil()
	return entity
}
