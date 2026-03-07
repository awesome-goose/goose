package testing

import (
	"reflect"
	"testing"
)

// ServiceTest provides utilities for testing services
type ServiceTest struct {
	t         *testing.T
	tHelper   *T
	service   any
	container *TestContainer
}

// NewServiceTest creates a new service test helper
func NewServiceTest(t *testing.T, service any, container *TestContainer) *ServiceTest {
	return &ServiceTest{
		t:         t,
		tHelper:   New(t),
		service:   service,
		container: container,
	}
}

// T returns the test helper for assertions
func (st *ServiceTest) T() *T {
	return st.tHelper
}

// Run method on the service with the given arguments
func (st *ServiceTest) Run(methodName string, args ...any) []any {
	st.t.Helper()

	// Inject dependencies
	st.container.Inject(st.service)

	// Find method
	serviceVal := reflect.ValueOf(st.service)
	method := serviceVal.MethodByName(methodName)
	if !method.IsValid() {
		st.t.Fatalf("Method %s not found on service", methodName)
		return nil
	}

	// Convert args to reflect.Value
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	// Call method
	outVal := method.Call(in)

	// Convert results to []any
	out := make([]any, len(outVal))
	for i, v := range outVal {
		out[i] = v.Interface()
	}
	return out
}
