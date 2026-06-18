package embed

import (
	"context"
	"sync"
)

// MockEmbedder is a configurable test double for the Embedder port. It records
// calls and delegates to EmbedFn. If EmbedFn is nil it returns zero-vectors.
//
// Calls is guarded by an internal mutex so it is safe to read Calls (via
// CallCount or Snapshot) from goroutines other than the one calling Embed.
type MockEmbedder struct {
	EmbedFn   func(inputs []string) ([][]float32, error)
	DimsValue int

	mu    sync.Mutex
	Calls [][]string // each entry is the inputs of one Embed call; use Snapshot() for concurrent access
}

// Embed records the inputs and returns EmbedFn's result (or zero-vectors).
func (m *MockEmbedder) Embed(_ context.Context, inputs []string) ([][]float32, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, append([]string(nil), inputs...))
	m.mu.Unlock()

	if m.EmbedFn != nil {
		return m.EmbedFn(inputs)
	}
	out := make([][]float32, len(inputs))
	for i := range out {
		out[i] = make([]float32, m.Dims())
	}
	return out, nil
}

// CallCount returns the number of Embed calls recorded so far. Thread-safe.
func (m *MockEmbedder) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Calls)
}

// Snapshot returns a shallow copy of the recorded call list. Thread-safe.
func (m *MockEmbedder) Snapshot() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([][]string, len(m.Calls))
	copy(out, m.Calls)
	return out
}

// Dims returns DimsValue, defaulting to 3072 when unset.
func (m *MockEmbedder) Dims() int {
	if m.DimsValue == 0 {
		return defaultDims
	}
	return m.DimsValue
}
