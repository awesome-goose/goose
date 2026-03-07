package testing

import (
	"os"
	"testing"
)

// Test level tags - use build tags or environment variables
const (
	TagUnit        = "unit"
	TagIntegration = "integration"
	TagE2E         = "e2e"
)

// SkipUnlessUnit skips unless running unit tests
func SkipUnlessUnit(t *testing.T) {
	if os.Getenv("TEST_LEVEL") != "" && os.Getenv("TEST_LEVEL") != TagUnit {
		t.Skip("Skipping unit test")
	}
}

// SkipUnlessIntegration skips unless running integration tests
func SkipUnlessIntegration(t *testing.T) {
	if os.Getenv("TEST_LEVEL") != TagIntegration && os.Getenv("TEST_LEVEL") != TagE2E {
		t.Skip("Skipping integration test - set TEST_LEVEL=integration")
	}
}

// SkipUnlessE2E skips unless running e2e tests
func SkipUnlessE2E(t *testing.T) {
	if os.Getenv("TEST_LEVEL") != TagE2E {
		t.Skip("Skipping e2e test - set TEST_LEVEL=e2e")
	}
}

// SkipInCI skips tests in CI environment
func SkipInCI(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}
}

// SkipShort skips if running in short mode
func SkipShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
}

// IsCI returns true if running in CI environment
func IsCI() bool {
	return os.Getenv("CI") != ""
}

// TestLevel returns the current test level
func TestLevel() string {
	level := os.Getenv("TEST_LEVEL")
	if level == "" {
		return TagUnit
	}
	return level
}
