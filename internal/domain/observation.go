package domain

// Source is a bitmask indicating which retrieval path found an observation.
type Source int

const (
	SourceFTS  Source = 1 // lexical FTS5 result
	SourceVec  Source = 2 // vector KNN result
	SourceBoth Source = 3 // found by both
)

// Observation is a live record from engram.db.
type Observation struct {
	ID        int64
	Title     string
	Content   string
	Project   string
	Scope     string
	Type      string
	TopicKey  string
	UpdatedAt string
}

// Ranked is an observation ID with a retrieval score and source attribution.
type Ranked struct {
	ID     int64
	Score  float64
	Source Source
}

// VecRow is a row to upsert into the vector store.
type VecRow struct {
	ObsID       int64
	Embedding   []float32
	ContentHash string
	Model       string
}

// SearchResult is a hydrated search hit returned to the caller.
type SearchResult struct {
	ID      int64
	Title   string
	Snippet string // first 200 chars of Content
	Score   float64
	Source  Source
}
