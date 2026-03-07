package testing

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

// T wraps *testing.T with fluent assertions
type T struct {
	t       *testing.T
	message string
}

// New creates a new test wrapper
func New(t *testing.T) *T {
	t.Helper()
	return &T{t: t}
}

// T returns the underlying *testing.T
func (t *T) T() *testing.T {
	return t.t
}

// Expect starts an assertion chain
func (t *T) Expect(actual any) *Assertion {
	t.t.Helper()
	return &Assertion{
		t:      t,
		actual: actual,
	}
}

// Assertion holds the actual value being tested
type Assertion struct {
	t       *T
	actual  any
	negated bool
}

// Not negates the next assertion
func (a *Assertion) Not() *Assertion {
	a.negated = !a.negated
	return a
}

// ToEqual asserts equality using reflect.DeepEqual
func (a *Assertion) ToEqual(expected any) *T {
	a.t.t.Helper()
	if a.negated {
		if reflect.DeepEqual(a.actual, expected) {
			a.t.fail(fmt.Sprintf("Expected values not to be equal, but they were: %#v", a.actual))
		}
	} else {
		if !reflect.DeepEqual(a.actual, expected) {
			a.t.fail(fmt.Sprintf("Expected values to be equal, but they were not.%s", diff(expected, a.actual)))
		}
	}
	return a.t
}

// ToDeepEqual asserts deep equality with detailed diff
func (a *Assertion) ToDeepEqual(expected any) *T {
	a.t.t.Helper()
	equal := reflect.DeepEqual(expected, a.actual)
	if a.negated {
		if equal {
			a.t.fail(fmt.Sprintf("Expected values not to be deeply equal, but they were: %#v", a.actual))
		}
	} else {
		if !equal {
			a.t.fail(fmt.Sprintf("Expected values to be deeply equal, but they were not.%s", deepDiff(expected, a.actual)))
		}
	}
	return a.t
}

// ToBe asserts identity (same pointer/value)
func (a *Assertion) ToBe(expected any) *T {
	a.t.t.Helper()
	if a.negated {
		if a.actual == expected {
			a.t.fail(fmt.Sprintf("Expected values not to be the same, but they were: %#v", a.actual))
		}
	} else {
		if a.actual != expected {
			a.t.fail(fmt.Sprintf("Expected values to be the same, but they were not.%s", diff(expected, a.actual)))
		}
	}
	return a.t
}

// ToBeNil asserts the value is nil
func (a *Assertion) ToBeNil() *T {
	a.t.t.Helper()
	if a.negated {
		if isNil(a.actual) {
			a.t.fail("Expected value not to be nil, but it was")
		}
	} else {
		if !isNil(a.actual) {
			a.t.fail(fmt.Sprintf("Expected value to be nil, but it was: %#v", a.actual))
		}
	}
	return a.t
}

// ToBeTrue asserts the value is true
func (a *Assertion) ToBeTrue() *T {
	a.t.t.Helper()
	isTrue := a.actual == true
	if a.negated {
		if isTrue {
			a.t.fail("Expected value not to be true, but it was")
		}
	} else {
		if !isTrue {
			a.t.fail("Expected value to be true, but it was not")
		}
	}
	return a.t
}

// ToBeFalse asserts the value is false
func (a *Assertion) ToBeFalse() *T {
	a.t.t.Helper()
	isFalse := a.actual == false
	if a.negated {
		if isFalse {
			a.t.fail("Expected value not to be false, but it was")
		}
	} else {
		if !isFalse {
			a.t.fail("Expected value to be false, but it was not")
		}
	}
	return a.t
}

// ToBeEmpty asserts the value is empty (zero length)
func (a *Assertion) ToBeEmpty() *T {
	a.t.t.Helper()
	l := length(a.actual)
	if a.negated {
		if l == 0 {
			a.t.fail("Expected value not to be empty, but it was")
		}
	} else {
		if l != 0 {
			a.t.fail(fmt.Sprintf("Expected value to be empty, but its length was %d", l))
		}
	}
	return a.t
}

// ToHaveLength asserts the value has the expected length
func (a *Assertion) ToHaveLength(expected int) *T {
	a.t.t.Helper()
	l := length(a.actual)
	if a.negated {
		if l == expected {
			a.t.fail(fmt.Sprintf("Expected length not to be %d, but it was", expected))
		}
	} else {
		if l != expected {
			a.t.fail(fmt.Sprintf("Expected length to be %d, but it was %d", expected, l))
		}
	}
	return a.t
}

// ToContain asserts the value contains the expected element
func (a *Assertion) ToContain(expected any) *T {
	a.t.t.Helper()
	if a.negated {
		if contains(a.actual, expected) {
			a.t.fail(fmt.Sprintf("Expected collection not to contain %#v, but it did", expected))
		}
	} else {
		if !contains(a.actual, expected) {
			a.t.fail(fmt.Sprintf("Expected collection to contain %#v, but it did not", expected))
		}
	}
	return a.t
}

// ToContainString asserts the string contains the expected substring
func (a *Assertion) ToContainString(expected string) *T {
	a.t.t.Helper()
	actualStr, ok := a.actual.(string)
	if !ok {
		a.t.fail("Actual value is not a string")
		return a.t
	}
	if a.negated {
		if strings.Contains(actualStr, expected) {
			a.t.fail(fmt.Sprintf("Expected string not to contain %#v, but it did", expected))
		}
	} else {
		if !strings.Contains(actualStr, expected) {
			a.t.fail(fmt.Sprintf("Expected string to contain %#v, but it did not", expected))
		}
	}
	return a.t
}

// ToHavePrefix asserts the string has the expected prefix
func (a *Assertion) ToHavePrefix(expected string) *T {
	a.t.t.Helper()
	actualStr, ok := a.actual.(string)
	if !ok {
		a.t.fail("Actual value is not a string")
		return a.t
	}
	if a.negated {
		if strings.HasPrefix(actualStr, expected) {
			a.t.fail(fmt.Sprintf("Expected string not to have prefix %#v, but it did", expected))
		}
	} else {
		if !strings.HasPrefix(actualStr, expected) {
			a.t.fail(fmt.Sprintf("Expected string to have prefix %#v, but it did not", expected))
		}
	}
	return a.t
}

// ToHaveSuffix asserts the string has the expected suffix
func (a *Assertion) ToHaveSuffix(expected string) *T {
	a.t.t.Helper()
	actualStr, ok := a.actual.(string)
	if !ok {
		a.t.fail("Actual value is not a string")
		return a.t
	}
	if a.negated {
		if strings.HasSuffix(actualStr, expected) {
			a.t.fail(fmt.Sprintf("Expected string not to have suffix %#v, but it did", expected))
		}
	} else {
		if !strings.HasSuffix(actualStr, expected) {
			a.t.fail(fmt.Sprintf("Expected string to have suffix %#v, but it did not", expected))
		}
	}
	return a.t
}

// ToBeGreaterThan asserts numeric comparison
func (a *Assertion) ToBeGreaterThan(expected any) *T {
	a.t.t.Helper()
	if cmp := compare(a.actual, expected); a.negated {
		if cmp > 0 {
			a.t.fail(fmt.Sprintf("Expected %#v not to be greater than %#v", a.actual, expected))
		}
	} else {
		if cmp <= 0 {
			a.t.fail(fmt.Sprintf("Expected %#v to be greater than %#v", a.actual, expected))
		}
	}
	return a.t
}

// ToBeLessThan asserts numeric comparison
func (a *Assertion) ToBeLessThan(expected any) *T {
	a.t.t.Helper()
	if cmp := compare(a.actual, expected); a.negated {
		if cmp < 0 {
			a.t.fail(fmt.Sprintf("Expected %#v not to be less than %#v", a.actual, expected))
		}
	} else {
		if cmp >= 0 {
			a.t.fail(fmt.Sprintf("Expected %#v to be less than %#v", a.actual, expected))
		}
	}
	return a.t
}

// ToBeGreaterOrEqual asserts numeric comparison
func (a *Assertion) ToBeGreaterOrEqual(expected any) *T {
	a.t.t.Helper()
	if cmp := compare(a.actual, expected); a.negated {
		if cmp >= 0 {
			a.t.fail(fmt.Sprintf("Expected %#v not to be greater or equal to %#v", a.actual, expected))
		}
	} else {
		if cmp < 0 {
			a.t.fail(fmt.Sprintf("Expected %#v to be greater or equal to %#v", a.actual, expected))
		}
	}
	return a.t
}

// ToBeLessOrEqual asserts numeric comparison
func (a *Assertion) ToBeLessOrEqual(expected any) *T {
	a.t.t.Helper()
	if cmp := compare(a.actual, expected); a.negated {
		if cmp <= 0 {
			a.t.fail(fmt.Sprintf("Expected %#v not to be less or equal to %#v", a.actual, expected))
		}
	} else {
		if cmp > 0 {
			a.t.fail(fmt.Sprintf("Expected %#v to be less or equal to %#v", a.actual, expected))
		}
	}
	return a.t
}

// ToBeBetween asserts value is between min and max (inclusive)
func (a *Assertion) ToBeBetween(min, max any) *T {
	a.t.t.Helper()
	cmpMin := compare(a.actual, min)
	cmpMax := compare(a.actual, max)
	if a.negated {
		if cmpMin >= 0 && cmpMax <= 0 {
			a.t.fail(fmt.Sprintf("Expected %#v not to be between %#v and %#v", a.actual, min, max))
		}
	} else {
		if cmpMin < 0 || cmpMax > 0 {
			a.t.fail(fmt.Sprintf("Expected %#v to be between %#v and %#v", a.actual, min, max))
		}
	}
	return a.t
}

// ToBeType asserts the value is of the expected type
func (a *Assertion) ToBeType(expected string) *T {
	a.t.t.Helper()
	actualType := reflect.TypeOf(a.actual).String()
	if a.negated {
		if actualType == expected {
			a.t.fail(fmt.Sprintf("Expected type not to be %s, but it was", expected))
		}
	} else {
		if actualType != expected {
			a.t.fail(fmt.Sprintf("Expected type to be %s, but it was %s", expected, actualType))
		}
	}
	return a.t
}

// ToImplement asserts the value implements the expected interface
func (a *Assertion) ToImplement(iface any) *T {
	a.t.t.Helper()
	ifaceType := reflect.TypeOf(iface).Elem()
	actualType := reflect.TypeOf(a.actual)
	implements := actualType.Implements(ifaceType)
	if a.negated {
		if implements {
			a.t.fail(fmt.Sprintf("Expected %#v not to implement %s", a.actual, ifaceType.String()))
		}
	} else {
		if !implements {
			a.t.fail(fmt.Sprintf("Expected %#v to implement %s", a.actual, ifaceType.String()))
		}
	}
	return a.t
}

// ToPanic asserts the function panics
func (a *Assertion) ToPanic() *T {
	a.t.t.Helper()
	fn, ok := a.actual.(func())
	if !ok {
		a.t.fail("Actual value is not a function")
		return a.t
	}

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		fn()
	}()

	if a.negated {
		if panicked {
			a.t.fail("Expected function not to panic, but it did")
		}
	} else {
		if !panicked {
			a.t.fail("Expected function to panic, but it did not")
		}
	}
	return a.t
}

// ToPanicWith asserts the function panics with the expected value
func (a *Assertion) ToPanicWith(expected any) *T {
	a.t.t.Helper()
	fn, ok := a.actual.(func())
	if !ok {
		a.t.fail("Actual value is not a function")
		return a.t
	}

	panicked := false
	var recovered any
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				recovered = r
			}
		}()
		fn()
	}()

	if a.negated {
		if panicked && reflect.DeepEqual(recovered, expected) {
			a.t.fail(fmt.Sprintf("Expected function not to panic with %#v, but it did", expected))
		}
	} else {
		if !panicked {
			a.t.fail("Expected function to panic, but it did not")
		} else if !reflect.DeepEqual(recovered, expected) {
			a.t.fail(fmt.Sprintf("Expected function to panic with %#v, but it panicked with %#v", expected, recovered))
		}
	}
	return a.t
}

// ToHaveKey asserts a map has the expected key
func (a *Assertion) ToHaveKey(key any) *T {
	a.t.t.Helper()
	v := reflect.ValueOf(a.actual)
	if v.Kind() != reflect.Map {
		a.t.fail("Actual value is not a map")
		return a.t
	}
	kv := reflect.ValueOf(key)
	hasKey := v.MapIndex(kv).IsValid()

	if a.negated {
		if hasKey {
			a.t.fail(fmt.Sprintf("Expected map not to have key %#v, but it did", key))
		}
	} else {
		if !hasKey {
			a.t.fail(fmt.Sprintf("Expected map to have key %#v, but it did not", key))
		}
	}
	return a.t
}

// ToHaveKeyValue asserts a map has the expected key-value pair
func (a *Assertion) ToHaveKeyValue(key, value any) *T {
	a.t.t.Helper()
	v := reflect.ValueOf(a.actual)
	if v.Kind() != reflect.Map {
		a.t.fail("Actual value is not a map")
		return a.t
	}
	kv := reflect.ValueOf(key)
	val := v.MapIndex(kv)

	if a.negated {
		if val.IsValid() && reflect.DeepEqual(val.Interface(), value) {
			a.t.fail(fmt.Sprintf("Expected map not to have key-value pair %#v: %#v, but it did", key, value))
		}
	} else {
		if !val.IsValid() {
			a.t.fail(fmt.Sprintf("Expected map to have key %#v, but it did not", key))
		} else if !reflect.DeepEqual(val.Interface(), value) {
			a.t.fail(fmt.Sprintf("Expected map to have value %#v for key %#v, but it had %#v", value, key, val.Interface()))
		}
	}
	return a.t
}

// ToMatchRegex asserts string matches regex
func (a *Assertion) ToMatchRegex(pattern string) *T {
	a.t.t.Helper()
	actualStr, ok := a.actual.(string)
	if !ok {
		a.t.fail("Actual value is not a string")
		return a.t
	}
	matched, err := regexp.MatchString(pattern, actualStr)
	if err != nil {
		a.t.fail(fmt.Sprintf("Invalid regex pattern: %s", pattern))
		return a.t
	}

	if a.negated {
		if matched {
			a.t.fail(fmt.Sprintf("Expected string not to match regex %#v, but it did", pattern))
		}
	} else {
		if !matched {
			a.t.fail(fmt.Sprintf("Expected string to match regex %#v, but it did not", pattern))
		}
	}
	return a.t
}

// fail records a test failure
func (t *T) fail(message string) {
	t.t.Helper()
	if t.message != "" {
		message = t.message + ": " + message
	}
	t.t.Error(message)
}

// FailNow fails immediately
func (t *T) FailNow(message string) {
	t.t.Helper()
	t.fail(message)
	t.t.FailNow()
}

// Require returns a wrapper that fails immediately on assertion failure
func (t *T) Require(actual any) *RequiredAssertion {
	t.t.Helper()
	return &RequiredAssertion{
		Assertion: Assertion{t: t, actual: actual},
	}
}

// RequiredAssertion wraps Assertion to fail immediately
type RequiredAssertion struct {
	Assertion
}

func (r *RequiredAssertion) fail(message string) {
	r.t.t.Helper()
	r.t.FailNow(message)
}

// ToEqual with immediate failure
func (r *RequiredAssertion) ToEqual(expected any) {
	r.t.t.Helper()
	if !reflect.DeepEqual(r.actual, expected) {
		r.fail(fmt.Sprintf("Expected values to be equal, but they were not.%s", diff(expected, r.actual)))
	}
}

// ToBeNil with immediate failure
func (r *RequiredAssertion) ToBeNil() {
	r.t.t.Helper()
	if !isNil(r.actual) {
		r.fail(fmt.Sprintf("Expected value to be nil, but it was: %#v", r.actual))
	}
}

// --- Helper Functions ---

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

func length(v any) int {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return rv.Len()
	}
	return -1
}

func contains(collection any, element any) bool {
	cv := reflect.ValueOf(collection)
	ev := reflect.ValueOf(element)

	switch cv.Kind() {
	case reflect.String:
		return strings.Contains(cv.String(), ev.String())
	case reflect.Map:
		keys := cv.MapKeys()
		for _, k := range keys {
			if reflect.DeepEqual(k.Interface(), element) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < cv.Len(); i++ {
			if reflect.DeepEqual(cv.Index(i).Interface(), element) {
				return true
			}
		}
	}
	return false
}

func compare(a, b any) int {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	// Handle different types if possible (e.g. int, float)
	if av.Type().ConvertibleTo(bv.Type()) {
		av = av.Convert(bv.Type())
	} else if bv.Type().ConvertibleTo(av.Type()) {
		bv = bv.Convert(av.Type())
	}

	if av.Type() != bv.Type() {
		return 0 // Cannot compare
	}

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if av.Int() > bv.Int() {
			return 1
		}
		if av.Int() < bv.Int() {
			return -1
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if av.Uint() > bv.Uint() {
			return 1
		}
		if av.Uint() < bv.Uint() {
			return -1
		}
	case reflect.Float32, reflect.Float64:
		if av.Float() > bv.Float() {
			return 1
		}
		if av.Float() < bv.Float() {
			return -1
		}
	case reflect.String:
		if av.String() > bv.String() {
			return 1
		}
		if av.String() < bv.String() {
			return -1
		}
	case reflect.Struct:
		if av.Type().String() == "time.Time" {
			at := av.Interface().(time.Time)
			bt := bv.Interface().(time.Time)
			if at.After(bt) {
				return 1
			}
			if at.Before(bt) {
				return -1
			}
		}
	}
	return 0
}

func diff(expected, actual any) string {
	// Simple diff output
	return fmt.Sprintf("\n  expected: %#v\n  actual:   %#v", expected, actual)
}

func deepDiff(expected, actual any) string {
	// More detailed diff for deep equality
	expectedStr := fmt.Sprintf("%#v", expected)
	actualStr := fmt.Sprintf("%#v", actual)
	if expectedStr == actualStr {
		// Types might be different even if string representation is same
		return fmt.Sprintf("\n  expected type: %T, value: %#v\n  actual type:   %T, value: %#v",
			expected, expected, actual, actual)
	}
	return fmt.Sprintf("\n  expected: %#v\n  actual:   %#v", expected, actual)
}

// Parallel marks test for parallel execution
func (t *T) Parallel() *T {
	t.t.Parallel()
	return t
}

// Run runs a subtest
func (t *T) Run(name string, fn func(t *T)) {
	t.t.Run(name, func(tt *testing.T) {
		fn(New(tt))
	})
}

// Skip skips the current test
func (t *T) Skip(reason ...string) {
	if len(reason) > 0 {
		t.t.Skip(strings.Join(reason, " "))
	} else {
		t.t.Skip()
	}
}

// SkipIf conditionally skips the test
func (t *T) SkipIf(condition bool, reason string) {
	if condition {
		t.t.Skip(reason)
	}
}

// --- Spy ---

// Spy wraps a real object and records calls
type Spy struct {
	Mock
	real any
}

// NewSpy creates a spy that wraps a real object
func NewSpy(t *testing.T, real any) *Spy {
	t.Helper()
	return &Spy{
		Mock: *NewMock(t),
		real: real,
	}
}

// CallReal invokes the real method and records the call
func (s *Spy) CallReal(method string, args ...any) []any {
	s.Called(method, args...)

	// Find the real method
	realVal := reflect.ValueOf(s.real)
	m := realVal.MethodByName(method)
	if !m.IsValid() {
		s.t.Fatalf("Method %s not found on real object", method)
		return nil
	}

	// Convert args to reflect.Value
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	// Call the real method
	outVal := m.Call(in)

	// Convert results to []any
	out := make([]any, len(outVal))
	for i, v := range outVal {
		out[i] = v.Interface()
	}
	return out
}

// --- Function Stub ---

// FuncStub creates a stub for a function variable
type FuncStub[T any] struct {
	target   *T
	original T
	restored bool
}

// StubFunc creates a function stub
func StubFunc[T any](target *T, replacement T) *FuncStub[T] {
	orig := *target
	*target = replacement
	return &FuncStub[T]{
		target:   target,
		original: orig,
	}
}

// Restore restores the original function
func (f *FuncStub[T]) Restore() {
	if f.restored {
		return
	}
	*f.target = f.original
	f.restored = true
}
