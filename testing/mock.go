package testing

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

// Mock provides a simple mock implementation
type Mock struct {
	t            *testing.T
	mu           sync.Mutex
	calls        []Call
	stubs        map[string]func(args ...any) []any
	expectations []Expectation
}

// Call represents a recorded function call
type Call struct {
	Method string
	Args   []any
	Time   int64
}

// Expectation represents an expected call
type Expectation struct {
	Method   string
	Args     []any
	Times    int
	actual   int
	Optional bool
}

// NewMock creates a new mock object
func NewMock(t *testing.T) *Mock {
	t.Helper()
	return &Mock{
		t:     t,
		stubs: make(map[string]func(args ...any) []any),
	}
}

// On sets up a return value for a method
func (m *Mock) On(method string, returns ...any) *Mock {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stubs[method] = func(args ...any) []any {
		return returns
	}
	return m
}

// Stub sets up a stub function for a method
func (m *Mock) Stub(method string, fn func(args ...any) []any) *Mock {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stubs[method] = fn
	return m
}

// ExpectCall adds an expectation for a method call
func (m *Mock) ExpectCall(method string, times int, args ...any) *Mock {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expectations = append(m.expectations, Expectation{
		Method: method,
		Args:   args,
		Times:  times,
	})
	return m
}

// Called records a method call and returns configured values
func (m *Mock) Called(method string, args ...any) []any {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, Call{
		Method: method,
		Args:   args,
		Time:   time.Now().UnixNano(),
	})

	for i := range m.expectations {
		exp := &m.expectations[i]
		if exp.Method == method && argsMatch(exp.Args, args) {
			exp.actual++
		}
	}

	if stub, ok := m.stubs[method]; ok {
		return stub(args...)
	}

	m.t.Logf("unexpected call to %s with args %v", method, args)
	return nil
}

// Verify checks all expectations were met
func (m *Mock) Verify() bool {
	m.t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	allMet := true
	for _, exp := range m.expectations {
		if !exp.Optional && exp.actual != exp.Times {
			m.t.Errorf("expected call to %s %d times, but was called %d times", exp.Method, exp.Times, exp.actual)
			allMet = false
		}
	}
	return allMet
}

// WasCalled checks if a method was called
func (m *Mock) WasCalled(method string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.calls {
		if call.Method == method {
			return true
		}
	}
	return false
}

// WasCalledWith checks if a method was called with specific args
func (m *Mock) WasCalledWith(method string, args ...any) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.calls {
		if call.Method == method && argsMatch(call.Args, args) {
			return true
		}
	}
	return false
}

// CallCount returns the number of times a method was called
func (m *Mock) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, call := range m.calls {
		if call.Method == method {
			count++
		}
	}
	return count
}

// GetCalls returns all calls to a method
func (m *Mock) GetCalls(method string) []Call {
	m.mu.Lock()
	defer m.mu.Unlock()
	var calls []Call
	for _, call := range m.calls {
		if call.Method == method {
			calls = append(calls, call)
		}
	}
	return calls
}

// Reset clears all recorded calls
func (m *Mock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
	m.expectations = nil
	m.stubs = make(map[string]func(args ...any) []any)
}

func argsMatch(expected, actual []any) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i, exp := range expected {
		if !reflect.DeepEqual(exp, actual[i]) {
			return false
		}
	}
	return true
}
