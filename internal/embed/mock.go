package embed

import "context"

// MockEmbedder is a configurable test double for the Embedder port. It records
// calls and delegates to EmbedFn. If EmbedFn is nil it returns zero-vectors.
type MockEmbedder struct {
	EmbedFn   func(inputs []string) ([][]float32, error)
	DimsValue int
	Calls     [][]string // each entry is the inputs of one Embed call
}

// Embed records the inputs and returns EmbedFn's result (or zero-vectors).
func (m *MockEmbedder) Embed(_ context.Context, inputs []string) ([][]float32, error) {
	m.Calls = append(m.Calls, append([]string(nil), inputs...))
	if m.EmbedFn != nil {
		return m.EmbedFn(inputs)
	}
	out := make([][]float32, len(inputs))
	for i := range out {
		out[i] = make([]float32, m.Dims())
	}
	return out, nil
}

// Dims returns DimsValue, defaulting to 3072 when unset.
func (m *MockEmbedder) Dims() int {
	if m.DimsValue == 0 {
		return defaultDims
	}
	return m.DimsValue
}
