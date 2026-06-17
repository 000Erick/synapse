# vector-store Specification

## Purpose

sqlite-vec persistence and K-Nearest-Neighbor search over 3072-dimensional
embeddings in `synapse.db`. This capability owns the `VectorStore` port
and its `sqlite-vec` CGO adapter.

## Requirements

### Requirement: Vector Schema Initialization

`synapse.db` MUST contain a `vec0` virtual table with exactly 3072
float32 dimensions. The schema MUST be created on first startup if it
does not exist. Subsequent starts MUST be idempotent (no error if schema
already exists).

#### Scenario: Schema created on first start

- GIVEN a blank `synapse.db` (new file)
- WHEN the vector store initializes
- THEN a `vec0` virtual table exists with 3072 float32 dimensions
- AND no error is returned

#### Scenario: Second start is idempotent

- GIVEN a `synapse.db` that already has the `vec0` table
- WHEN the vector store initializes again
- THEN no error is returned
- AND the existing vectors are intact

### Requirement: Vector Upsert

The vector store MUST support inserting or replacing a vector keyed by
`observation_id` (integer). If the `observation_id` already exists, the
vector MUST be replaced (not duplicated). The vector dimension MUST equal
3072; mismatched dimensions MUST return an error.

#### Scenario: Insert a new vector

- GIVEN an empty `vec0` table
- WHEN a 3072-dim vector is upserted with `observation_id=1`
- THEN the vector count is 1

#### Scenario: Re-upsert replaces existing vector without duplication

- GIVEN a vector with `observation_id=1` already stored
- WHEN the same `observation_id=1` is upserted with a different vector
- THEN the vector count remains 1
- AND the stored vector matches the new one

#### Scenario: Wrong dimension is rejected

- GIVEN the `vec0` table expects 3072 dims
- WHEN a 1536-dim vector is upserted
- THEN an error is returned describing the dimension mismatch
- AND the table contents are unchanged

### Requirement: KNN Query

The vector store MUST support K-Nearest-Neighbor search given a query
vector (3072 dims) and a limit `k`. Results MUST be returned as
`(observation_id, distance)` pairs ordered by ascending distance (closest first).

#### Scenario: KNN returns closest vectors

- GIVEN 5 vectors in `vec0`, vector for id=3 is closest to the query
- WHEN KNN search is called with `k=3`
- THEN 3 results are returned
- AND `observation_id=3` appears first (smallest distance)

#### Scenario: KNN with k larger than stored count returns all stored

- GIVEN only 2 vectors in `vec0`
- WHEN KNN search is called with `k=10`
- THEN 2 results are returned without error

### Requirement: Vector Deletion

The vector store MUST support deleting one or more vectors by
`observation_id` in a single transaction. Deleting a non-existent id
MUST be a no-op (no error).

#### Scenario: Delete existing vectors in transaction

- GIVEN 3 vectors with ids 1, 2, 3
- WHEN ids 1 and 2 are deleted in a single transaction
- THEN only the vector for id=3 remains

#### Scenario: Delete non-existent id is a no-op

- GIVEN an empty `vec0` table
- WHEN delete is called with id=99
- THEN no error is returned

### Requirement: Vector Count and Stats

The vector store MUST expose a method returning the total count of stored
vectors (used by `synapse_stats`).

#### Scenario: Count returns correct total

- GIVEN 7 vectors in `vec0`
- WHEN count is called
- THEN 7 is returned

### Requirement: Test Isolation

All `VectorStore` tests MUST use `t.TempDir()` for the DB file or an
in-memory equivalent. No test MUST require a real `synapse.db`.

#### Scenario: VectorStore tests run without real synapse.db

- GIVEN `go test ./...` is run
- WHEN no `SYNAPSE_DB_PATH` pointing to a permanent file is set
- THEN all vector-store tests pass using temp SQLite instances
