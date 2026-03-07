package testing

import (
	"testing"
	"time"
)

// Benchmark wraps *testing.B with helpers
type Benchmark struct {
	b *testing.B
}

// NewBenchmark creates a new benchmark wrapper
func NewBenchmark(b *testing.B) *Benchmark {
	return &Benchmark{b: b}
}

// ResetTimer resets the benchmark timer
func (b *Benchmark) ResetTimer() *Benchmark {
	b.b.ResetTimer()
	return b
}

// StopTimer stops the benchmark timer
func (b *Benchmark) StopTimer() *Benchmark {
	b.b.StopTimer()
	return b
}

// StartTimer starts the benchmark timer
func (b *Benchmark) StartTimer() *Benchmark {
	b.b.StartTimer()
	return b
}

// SetBytes records the number of bytes processed
func (b *Benchmark) SetBytes(n int64) *Benchmark {
	b.b.SetBytes(n)
	return b
}

// SetParallelism sets the number of goroutines for RunParallel
func (b *Benchmark) SetParallelism(p int) *Benchmark {
	b.b.SetParallelism(p)
	return b
}

// N returns the number of iterations
func (b *Benchmark) N() int {
	return b.b.N
}

// Run runs a sub-benchmark
func (b *Benchmark) Run(name string, fn func(b *Benchmark)) {
	b.b.Run(name, func(bb *testing.B) {
		fn(NewBenchmark(bb))
	})
}

// RunParallel runs fn in parallel
func (b *Benchmark) RunParallel(fn func(pb *testing.PB)) {
	b.b.RunParallel(fn)
}

// ReportAllocs enables allocation reporting
func (b *Benchmark) ReportAllocs() *Benchmark {
	b.b.ReportAllocs()
	return b
}

// ReportMetric reports a custom metric
func (b *Benchmark) ReportMetric(n float64, unit string) *Benchmark {
	b.b.ReportMetric(n, unit)
	return b
}

// --- Timer utilities ---

// Timer measures execution time
type Timer struct {
	start time.Time
	end   time.Time
}

// StartTimer starts a timer
func StartTimer() *Timer {
	return &Timer{start: time.Now()}
}

// Stop stops the timer
func (t *Timer) Stop() time.Duration {
	t.end = time.Now()
	return t.Duration()
}

// Duration returns the elapsed duration
func (t *Timer) Duration() time.Duration {
	if t.end.IsZero() {
		return time.Since(t.start)
	}
	return t.end.Sub(t.start)
}

// Elapsed returns the elapsed time as string
func (t *Timer) Elapsed() string {
	return t.Duration().String()
}
