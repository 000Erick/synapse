package embed

import (
	"context"
)

const defaultDims = 3072

// NoopEmbedder is a stub that returns zero-vectors. Used for tests and
// as a placeholder before the real OpenAI embedder is wired.
type NoopEmbedder struct{}

// Embed returns a slice of zero-vectors of length 3072, one per input.
func (n *NoopEmbedder) Embed(_ context.Context, inputs []string) ([][]float32, error) {
	result := make([][]float32, len(inputs))
	for i := range result {
		result[i] = make([]float32, defaultDims)
	}
	return result, nil
}

// Dims returns the embedding dimension (3072).
func (n *NoopEmbedder) Dims() int {
	return defaultDims
}
