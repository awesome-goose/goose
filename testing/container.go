package testing

import (
	"reflect"
	"sync"
)

// TestContainer provides a simple DI container for testing
type TestContainer struct {
	mu      sync.Mutex
	entries map[reflect.Type]any
}

// NewTestContainer creates a new test container
func NewTestContainer() *TestContainer {
	return &TestContainer{
		entries: make(map[reflect.Type]any),
	}
}

// Register a dependency
func (c *TestContainer) Register(dependency any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[reflect.TypeOf(dependency)] = dependency
}

// RegisterAs registers a dependency with a specific interface type
func (c *TestContainer) RegisterAs(iface any, dependency any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[reflect.TypeOf(iface).Elem()] = dependency
}

// Resolve a dependency
func (c *TestContainer) Resolve(target any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	targetVal := reflect.ValueOf(target).Elem()
	targetType := targetVal.Type()

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if dep, ok := c.entries[field.Type]; ok {
			targetVal.Field(i).Set(reflect.ValueOf(dep))
		}
	}
}

// Inject dependencies into a struct
func (c *TestContainer) Inject(target any) {
	c.Resolve(target)
}
