# engram-reader Specification

## Purpose

Read-only access to Engram observations and FTS5 search. This capability
owns the `EngramReader` port and its SQLite adapter. It MUST NEVER write
to `engram.db`.

## Requirements

### Requirement: Read-Only Database Contract

`engram.db` MUST be opened with the URI parameter `mode=ro` (e.g.
`file:///path/engram.db?mode=ro`). Any attempt to execute a write
statement (INSERT, UPDATE, DELETE, DDL) MUST be rejected by SQLite before
it reaches the application layer.

#### Scenario: engram.db opened read-only

- GIVEN `engram.db` exists and is opened by the reader
- WHEN the SQLite connection is inspected
- THEN the connection URI contains `mode=ro`
- AND executing `INSERT INTO observations ...` on that connection returns an error

#### Scenario: Zero writes to engram.db during any tool call

- GIVEN a temp `engram.db` seeded with 3 observations (via a separate write connection)
- WHEN each of the five MCP tools is called in sequence
- THEN the `engram.db` file's modification time does not change
- AND the WAL file for `engram.db` contains no new frames

### Requirement: Live Observation Read

The reader MUST return only observations where `deleted_at IS NULL`.
It MUST select exactly: `id`, `title`, `content`, `project`, `scope`,
`type`, `topic_key`, `updated_at`. It MUST NOT use `SELECT *`.

#### Scenario: Returns only live observations

- GIVEN `engram.db` contains 5 observations: 4 live (`deleted_at IS NULL`) and 1 soft-deleted
- WHEN the reader fetches all live observations
- THEN exactly 4 observations are returned
- AND the soft-deleted observation is absent

#### Scenario: Returns correct columns

- GIVEN a live observation with all fields populated
- WHEN the reader returns it
- THEN the returned struct has: `ID`, `Title`, `Content`, `Project`, `Scope`, `Type`, `TopicKey`, `UpdatedAt`
- AND no additional raw SQL columns are exposed to the domain

### Requirement: FTS5 Lexical Search

The reader MUST support FTS5 search against `observations_fts` using
MATCH syntax. Results MUST be returned in BM25 rank order (most relevant
first). Search MUST be constrained to live observations (`deleted_at IS NULL`).

#### Scenario: FTS5 returns ranked matches for a keyword query

- GIVEN `engram.db` contains 3 live observations; one contains "architecture" in title, another in content
- WHEN the reader runs FTS5 search for `"architecture"`
- THEN at least 2 results are returned, ordered by BM25 score descending
- AND any soft-deleted observation is not included

#### Scenario: FTS5 returns empty list for unmatched query

- GIVEN `engram.db` contains no observations matching `"xyzunmatched"`
- WHEN the reader runs FTS5 search for `"xyzunmatched"`
- THEN an empty list is returned without error

#### Scenario: FTS5 search limit is honored

- GIVEN `engram.db` contains 20 observations all matching `"note"`
- WHEN the reader runs FTS5 search with `limit=5`
- THEN exactly 5 results are returned

### Requirement: Observation Count

The reader MUST expose a method to return the total count of live
observations (used by `synapse_health` and `synapse_stats`).

#### Scenario: Count returns correct live total

- GIVEN `engram.db` with 10 live and 2 soft-deleted observations
- WHEN the reader counts live observations
- THEN the result is `10`

### Requirement: Test Isolation

All `EngramReader` tests MUST use a temp SQLite DB (via `t.TempDir()`).
No test MUST reference the real `~/.engram/engram.db`.

#### Scenario: Tests run without real engram.db

- GIVEN the test suite runs with `go test ./...`
- WHEN no `ENGRAM_DB_PATH` pointing to a real DB is set
- THEN all `engram-reader` tests pass using temp SQLite instances
