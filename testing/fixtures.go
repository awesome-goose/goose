package testing

import (
	"math/rand"
	"time"
)

// Fixture provides test data fixtures
type Fixture struct {
	rand *rand.Rand
}

// NewFixture creates a new fixture generator
func NewFixture() *Fixture {
	return &Fixture{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// String generates a random string of a given length
func (f *Fixture) String(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[f.rand.Intn(len(charset))]
	}
	return string(b)
}

// Int generates a random integer within a given range
func (f *Fixture) Int(min, max int) int {
	return f.rand.Intn(max-min+1) + min
}

// Email generates a random email address
func (f *Fixture) Email() string {
	return f.String(10) + "@" + f.String(5) + ".com"
}

// Bool generates a random boolean
func (f *Fixture) Bool() bool {
	return f.rand.Intn(2) == 1
}
