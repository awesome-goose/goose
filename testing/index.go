// Package testing provides a comprehensive testing framework for Go applications.
// It combines both general-purpose testing utilities and Goose-specific testing helpers
// into a single unified package.
//
// This package provides:
//   - Fluent assertions with chainable API
//   - Mock/stub utilities for dependency isolation
//   - Test suite management with setup/teardown
//   - HTTP testing utilities
//   - Benchmark helpers
//   - Test tagging for unit/integration/e2e tests
//   - Goose-specific context, controller, service, and module testing
//
// Example usage:
//
//	func TestExample(t *testing.T) {
//		test := testing.New(t)
//		test.Expect("hello").ToEqual("hello")
//		test.Expect(42).ToBeGreaterThan(10)
//	}
//
//	func TestGooseService(t *testing.T) {
//		gt := testing.NewGooseTest(t)
//		ctx := gt.Context()
//		ctx.MockRequest().WithMethod(types.GET).WithPaths("users", "123")
//		// ... test your service
//	}
package testing
