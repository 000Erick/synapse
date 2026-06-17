# hybrid-search Specification

## Purpose

`synapse_search` MCP tool: RRF fusion of Engram FTS5 lexical results and
sqlite-vec KNN vector results. Returns ranked observations with metadata
and source attribution.

## Requirements

### Requirement: Hybrid Search Input

`synapse_search` MUST accept a `query` string and an optional `limit`
integer (default 10, maximum 50). The query MUST NOT be empty; an empty
query MUST return an error.

#### Scenario: Empty query returns error

- GIVEN a valid server setup
- WHEN `synapse_search` is called with `query=""`
- THEN an error is returned: `"query must not be empty"`

#### Scenario: Default limit is 10

- GIVEN 20 matching observations in both FTS5 and vector results
- WHEN `synapse_search` is called with `query="test"` and no `limit`
- THEN at most 10 results are returned

#### Scenario: Custom limit is honored

- GIVEN enough matching observations
- WHEN `synapse_search` is called with `query="test"` and `limit=5`
- THEN at most 5 results are returned

### Requirement: FTS5 Lexical Retrieval

The search MUST execute an FTS5 MATCH query against `observations_fts` in
`engram.db` (read-only). Results MUST be constrained to live observations.
The FTS5 query MUST run regardless of whether `OPENAI_API_KEY` is set.

#### Scenario: FTS5 runs without API key

- GIVEN `OPENAI_API_KEY` is unset
- WHEN `synapse_search` is called with `query="architecture"`
- THEN the FTS5 leg runs and returns lexical matches
- AND the vector leg is skipped with a note in the result
- AND no error prevents the tool from returning results

### Requirement: Vector KNN Retrieval

When `OPENAI_API_KEY` is set, the search MUST embed the query via the
`Embedder` and run a KNN query against `synapse.db` (retrieving up to
`limit × 2` candidates before fusion). If the query embedding fails,
the search MUST fall back to FTS5-only results and MUST NOT return an
error to the client.

#### Scenario: KNN retrieval runs when API key is present

- GIVEN `OPENAI_API_KEY` is set and `MockEmbedder` returns a query vector
- WHEN `synapse_search` is called
- THEN the embedder is called once with the query string
- AND KNN results are retrieved from `synapse.db`

#### Scenario: Embedding failure falls back to FTS5-only

- GIVEN `MockEmbedder` returns an error on the query
- WHEN `synapse_search` is called
- THEN FTS5 results are returned
- AND the response includes a warning field: `"vector_unavailable": true`

### Requirement: Reciprocal Rank Fusion (RRF k=60)

The system MUST fuse FTS5 and KNN result lists using Reciprocal Rank
Fusion with constant `k=60`. The RRF score for observation `d` is:
`score(d) = Σ 1/(k + rank_i(d))` where `rank_i` is the 1-based rank
in each source list. Observations appearing in both lists receive
contributions from both sources.

#### Scenario: RRF fuses two ranked lists correctly

- GIVEN FTS5 list: [A(rank 1), B(rank 2), C(rank 3)]
- AND KNN list:   [B(rank 1), A(rank 2), D(rank 3)]
- WHEN RRF k=60 is applied
- THEN B's score = 1/61 + 1/62 ≈ 0.03255
- AND A's score  = 1/61 + 1/62 ≈ 0.03255  (tied — tiebreak by fts rank)
- AND C and D each have score = 1/63 ≈ 0.01587
- AND the fused result is ordered by descending RRF score

#### Scenario: Observation in only one source gets partial RRF score

- GIVEN observation E appears only in FTS5 at rank 1 (not in KNN)
- WHEN RRF is applied with k=60
- THEN E's score = 1/61 ≈ 0.01639

### Requirement: Cross-Language Semantic Recall

The vector search MUST enable cross-language and synonym recall: a query
in one language MUST surface observations whose content is in a different
language when the semantic meaning matches.

#### Scenario: Spanish query surfaces English observations (acceptance test)

- GIVEN `engram.db` contains an observation with title "teacher" and content in English
- AND `synapse.db` has a vector for that observation computed from `text-embedding-3-large`
- AND a `MockEmbedder` returns a vector semantically close to "teacher" for the query "profesor"
- WHEN `synapse_search` is called with `query="profesor"`
- THEN the "teacher" observation appears in the top 5 results
- AND its `source` field is `"vec"` or `"both"`

### Requirement: Result Shape

Each result MUST include: `id` (int), `title` (string), `snippet` (first
200 chars of `content`), `score` (float, RRF score), `source` (`"fts"` |
`"vec"` | `"both"`).

#### Scenario: Result fields are all present

- GIVEN a search returns 1 result
- WHEN the result is inspected
- THEN it contains `id`, `title`, `snippet`, `score`, and `source`
- AND `snippet` is at most 200 characters
- AND `source` is one of `"fts"`, `"vec"`, `"both"`

### Requirement: Test Isolation

All `hybrid-search` tests MUST use temp SQLite DBs and `MockEmbedder`.
The RRF fusion function MUST be testable as a pure function with no DB
dependency.

#### Scenario: RRF function tested with no DB

- GIVEN two in-memory ranked lists (no SQLite)
- WHEN the RRF function is called
- THEN it returns the correct fused order
- AND the test does not open any DB connection
