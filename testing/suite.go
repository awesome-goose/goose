package testing

import (
	"reflect"
	"strings"
	"testing"
)

// Suite is a base struct for test suites
type Suite struct {
	t        *testing.T
	T        *T
	setup    func()
	teardown func()
}

// SuiteRunner runs a test suite
type SuiteRunner struct {
	t     *testing.T
	suite any
}

// NewSuiteRunner creates a new suite runner
func NewSuiteRunner(t *testing.T, suite any) *SuiteRunner {
	t.Helper()
	return &SuiteRunner{t: t, suite: suite}
}

// Run executes all test methods in the suite
func (r *SuiteRunner) Run() {
	r.t.Helper()
	suiteVal := reflect.ValueOf(r.suite)
	suiteType := suiteVal.Type()

	// Set T on suite
	if s, ok := r.suite.(Suiter); ok {
		s.SetT(r.t)
	}

	// Run setup suite
	if s, ok := r.suite.(SetupSuiteFunc); ok {
		s.SetupSuite()
	}
	defer func() {
		// Run teardown suite
		if s, ok := r.suite.(TeardownSuiteFunc); ok {
			s.TeardownSuite()
		}
	}()

	for i := 0; i < suiteType.NumMethod(); i++ {
		method := suiteType.Method(i)
		if strings.HasPrefix(method.Name, "Test") {
			r.t.Run(method.Name, func(t *testing.T) {
				// Run setup test
				if s, ok := r.suite.(SetupTestFunc); ok {
					s.SetupTest()
				}
				defer func() {
					// Run teardown test
					if s, ok := r.suite.(TeardownTestFunc); ok {
						s.TeardownTest()
					}
				}()
				method.Func.Call([]reflect.Value{suiteVal})
			})
		}
	}
}

// Suiter interface for test suites
type Suiter interface {
	SetT(t *testing.T)
}

// SetT sets the testing.T on the suite
func (s *Suite) SetT(t *testing.T) {
	s.t = t
	s.T = New(t)
}

// SetupSuite runs once before all tests
type SetupSuiteFunc interface {
	SetupSuite()
}

// TeardownSuite runs once after all tests
type TeardownSuiteFunc interface {
	TeardownSuite()
}

// SetupTest runs before each test
type SetupTestFunc interface {
	SetupTest()
}

// TeardownTest runs after each test
type TeardownTestFunc interface {
	TeardownTest()
}

// RunSuite runs a test suite
func RunSuite(t *testing.T, suite Suiter) {
	t.Helper()
	NewSuiteRunner(t, suite).Run()
}

// --- Table-Driven Tests ---

// TableTest represents a table-driven test case
type TableTest[I any] struct {
	Name    string
	Input   I
	Test    func(t *T, input I)
	WantErr bool
	Skip    bool
}

// RunTable executes a table-driven test
func RunTable[I any](t *testing.T, tests []TableTest[I], fn func(t *T, tc TableTest[I])) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Skip {
				t.Skip()
			}
			fn(New(t), tc)
		})
	}
}
