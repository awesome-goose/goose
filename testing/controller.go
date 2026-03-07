package testing

import (
	"reflect"
	"testing"
)

// ControllerTest provides utilities for testing controllers
type ControllerTest struct {
	t          *testing.T
	tHelper    *T
	controller any
	container  *TestContainer
}

// NewControllerTest creates a new controller test helper
func NewControllerTest(t *testing.T, controller any, container *TestContainer) *ControllerTest {
	return &ControllerTest{
		t:          t,
		tHelper:    New(t),
		controller: controller,
		container:  container,
	}
}

// T returns the test helper for assertions
func (ct *ControllerTest) T() *T {
	return ct.tHelper
}

// Run method on the controller with the given context
func (ct *ControllerTest) Run(methodName string, ctx *MockContext) {
	ct.t.Helper()

	// Inject dependencies
	ct.container.Inject(ct.controller)

	// Find method
	controllerVal := reflect.ValueOf(ct.controller)
	method := controllerVal.MethodByName(methodName)
	if !method.IsValid() {
		ct.t.Fatalf("Method %s not found on controller", methodName)
		return
	}

	// Call method
	method.Call([]reflect.Value{reflect.ValueOf(ctx)})
}
