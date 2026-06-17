# vector-sync Specification

## Purpose

`synapse_sync` MCP tool: detects and removes orphan vectors — vectors in
`synapse.db` whose `observation_id` no longer exists in `engram.db` or
has `deleted_at` set. Live vectors MUST NOT be removed.

## Requirements

### Requirement: Orphan Detection

The system MUST define an orphan as any vector in `synapse.db` whose
`observation_id` either (a) does not appear in `engram.db.observations`,
or (b) appears with `deleted_at IS NOT NULL`. This check MUST query
`engram.db` read-only.

#### Scenario: Identifies orphans for deleted observations

- GIVEN `synapse.db` has vectors for ids [1, 2, 3]
- AND `engram.db` has: id=1 (live), id=2 (soft-deleted), id=3 (missing entirely)
- WHEN orphan detection runs
- THEN ids 2 and 3 are identified as orphans
- AND id=1 is not an orphan

#### Scenario: No orphans when all vectors map to live observations

- GIVEN `synapse.db` has vectors for ids [1, 2]
- AND `engram.db` has both as live
- WHEN orphan detection runs
- THEN 0 orphans are found

### Requirement: Transactional Orphan Deletion

Orphan vectors MUST be deleted from `synapse.db` in a single
transaction. If the transaction fails, no vectors MUST be deleted (atomic
all-or-nothing). Live vectors MUST NOT be deleted under any circumstance.

#### Scenario: Orphans deleted in one transaction

- GIVEN 2 orphan vectors (ids 2, 3) and 1 live vector (id 1)
- WHEN `synapse_sync` is called
- THEN ids 2 and 3 are deleted from `synapse.db`
- AND id 1 remains

#### Scenario: Transaction rollback on failure leaves all vectors intact

- GIVEN 2 orphan vectors exist
- AND the delete transaction is forced to fail (e.g. DB locked mock)
- WHEN `synapse_sync` is called
- THEN no vectors are deleted
- AND an error is returned describing the failure

### Requirement: engram.db Read-Only During Sync

`synapse_sync` MUST query `engram.db` exclusively through the
`EngramReader` port (mode=ro). It MUST NOT write to `engram.db`.

#### Scenario: engram.db not modified during sync

- GIVEN a temp `engram.db` opened read-only
- WHEN `synapse_sync` runs
- THEN `engram.db` modification time is unchanged

### Requirement: Sync Response

`synapse_sync` MUST return a JSON response with `removed` (count of
deleted orphan vectors) and `live_preserved` (count of vectors kept).

#### Scenario: Response counts are correct

- GIVEN 3 orphans and 5 live vectors
- WHEN `synapse_sync` is called
- THEN the response is `{"removed": 3, "live_preserved": 5}`

#### Scenario: No orphans — zero removed

- GIVEN 0 orphans and 4 live vectors
- WHEN `synapse_sync` is called
- THEN the response is `{"removed": 0, "live_preserved": 4}`

### Requirement: Test Isolation

All `vector-sync` tests MUST use temp SQLite DBs via `t.TempDir()`.
Tests MUST NOT read from real `engram.db` or `synapse.db`.

#### Scenario: Sync tests run without real DBs

- GIVEN both `engram.db` and `synapse.db` are temp in-memory instances
- WHEN `go test ./...` is run
- THEN all sync tests pass without accessing real DB files
