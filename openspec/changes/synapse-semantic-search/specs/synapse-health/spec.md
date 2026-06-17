# synapse-health Specification

## Purpose

`synapse_health` and `synapse_stats` MCP tools plus cross-cutting
constraints: read-only guarantee, configuration behavior, and strict
test-isolation requirements that apply to every capability.

## Requirements

### Requirement: Health Check Tool (synapse_health)

`synapse_health` MUST report liveness, DB reachability for both
`engram.db` (read-only) and `synapse.db`, and return version/path info.
It MUST work even when `OPENAI_API_KEY` is absent.

#### Scenario: Both DBs reachable — healthy response

- GIVEN both `engram.db` (read-only) and `synapse.db` are reachable
- WHEN `synapse_health` is called
- THEN response contains `{"status": "ok", "engram_reachable": true, "synapse_reachable": true, "engram_path": "...", "synapse_path": "...", "version": "..."}`

#### Scenario: engram.db unreachable — partial health

- GIVEN `ENGRAM_DB_PATH` points to a non-existent file
- WHEN `synapse_health` is called
- THEN response contains `{"status": "degraded", "engram_reachable": false}`
- AND the tool does NOT return an MCP-level error (it reports health inline)

#### Scenario: Health works without OPENAI_API_KEY

- GIVEN `OPENAI_API_KEY` is unset
- WHEN `synapse_health` is called
- THEN response contains `{"status": "ok"}` (assuming both DBs reachable)
- AND no error about missing API key appears in the response

### Requirement: Stats Tool (synapse_stats)

`synapse_stats` MUST return: total live observations count (from
`engram.db`), total vector count (from `synapse.db`), orphan count,
embedding dimensions (3072), and model name.

#### Scenario: Stats returns all fields

- GIVEN `engram.db` has 10 live obs, `synapse.db` has 8 vectors (2 orphans)
- WHEN `synapse_stats` is called
- THEN response contains:
  `{"live_observations": 10, "vectors": 8, "orphans": 2, "dims": 3072, "model": "text-embedding-3-large"}`

#### Scenario: Stats with zero vectors

- GIVEN `synapse.db` is empty and `engram.db` has 5 live obs
- WHEN `synapse_stats` is called
- THEN `{"live_observations": 5, "vectors": 0, "orphans": 0, "dims": 3072, "model": "text-embedding-3-large"}`

### Requirement: Read-Only Guarantee (Cross-Cutting)

`engram.db` MUST be opened with `file:...?mode=ro` in ALL code paths.
No SQL write statement (INSERT, UPDATE, DELETE, CREATE, DROP, ALTER) on
the `engram.db` connection is permitted. This applies to every MCP tool,
every background operation, and every test that uses the engram adapter.

#### Scenario: Zero writes to engram.db across all five tools

- GIVEN a temp `engram.db` seeded via a separate write connection, then closed
- WHEN all five MCP tools (`health`, `stats`, `backfill`, `search`, `sync`) are called in sequence
- THEN a subsequent read of the WAL file shows no new frames from Synapse
- AND the `engram.db` modification time is unchanged throughout

### Requirement: Configuration Defaults and Error Behavior (Cross-Cutting)

All four env variables MUST have defined defaults or error behavior:

| Variable | When absent | Behavior |
|---|---|---|
| `OPENAI_API_KEY` | Tools needing it called | Return error: `"OPENAI_API_KEY is not set"` |
| `OPENAI_EMBED_MODEL` | Not set | Default to `"text-embedding-3-large"` |
| `SYNAPSE_DB_PATH` | Not set | Default to `~/.synapse/synapse.db` |
| `ENGRAM_DB_PATH` | Not set | Default to `~/.engram/engram.db` |

#### Scenario: All defaults applied when no env vars set

- GIVEN no environment variables are set
- WHEN the server starts and `synapse_health` is called
- THEN the health response shows `engram_path: "~/.engram/engram.db"` and `synapse_path: "~/.synapse/synapse.db"`
- AND `synapse_stats` returns `model: "text-embedding-3-large"`

### Requirement: Strict TDD — No Real Network or DB in Tests (Cross-Cutting)

The test suite MUST pass without a real `OPENAI_API_KEY`, without a real
`~/.engram/engram.db`, and without a real `~/.synapse/synapse.db`.
Every test requiring SQLite MUST use `t.TempDir()`. Every test requiring
embedding MUST use `MockEmbedder`. Integration tests that would require
real external services MUST be skippable with `go test -short`.

#### Scenario: Full test suite passes with no real deps

- GIVEN `OPENAI_API_KEY` is unset, real DBs are absent
- WHEN `go test ./...` is run
- THEN all tests pass
- AND no real HTTP call to `api.openai.com` is made

#### Scenario: Integration tests are skippable in -short mode

- GIVEN a test uses a real SQLite file (not in-memory)
- WHEN `go test -short ./...` is run
- THEN that test is skipped
- AND the test binary exits cleanly
