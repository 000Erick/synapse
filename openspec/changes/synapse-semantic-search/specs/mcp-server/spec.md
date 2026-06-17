# mcp-server Specification

## Purpose

stdio MCP server lifecycle, tool registration, and environment-driven
configuration for the Synapse binary. This capability owns the process
entry-point, env parsing, and MCP transport.

## Requirements

### Requirement: Stdio MCP Server Lifecycle

The server MUST start, register all five tools, and serve requests over
stdio using the official `modelcontextprotocol/go-sdk` transport.
Startup MUST complete without writing to `engram.db`.
On a fatal initialization error (e.g. unable to open `synapse.db`), the
server MUST log the error and exit with a non-zero status code.

#### Scenario: Server starts and lists tools

- GIVEN a valid environment with `ENGRAM_DB_PATH` and `SYNAPSE_DB_PATH`
- WHEN the binary starts
- THEN all five tools (`synapse_health`, `synapse_stats`, `synapse_backfill`, `synapse_search`, `synapse_sync`) are registered and visible via MCP tool listing
- AND the process remains listening on stdin/stdout

#### Scenario: Fatal initialization failure exits cleanly

- GIVEN `SYNAPSE_DB_PATH` points to a non-writable directory
- WHEN the binary starts
- THEN it logs a descriptive error message to stderr
- AND exits with status code `1`

### Requirement: Environment Configuration

The server MUST read configuration exclusively from environment variables
at startup. It MUST NOT read files other than the two SQLite databases.

| Variable | Required | Default |
|---|---|---|
| `OPENAI_API_KEY` | Required for backfill/search | none — error if missing and those tools are called |
| `OPENAI_EMBED_MODEL` | Optional | `text-embedding-3-large` |
| `SYNAPSE_DB_PATH` | Optional | `~/.synapse/synapse.db` |
| `ENGRAM_DB_PATH` | Optional | `~/.engram/engram.db` |

#### Scenario: Default paths are used when env vars are absent

- GIVEN neither `SYNAPSE_DB_PATH` nor `ENGRAM_DB_PATH` are set
- WHEN the server starts
- THEN it attempts to open `~/.synapse/synapse.db` and `~/.engram/engram.db`

#### Scenario: Explicit paths override defaults

- GIVEN `SYNAPSE_DB_PATH=/tmp/s.db` and `ENGRAM_DB_PATH=/tmp/e.db`
- WHEN the server starts
- THEN it opens those exact paths (verified by `synapse_health` response)

#### Scenario: Missing OPENAI_API_KEY does not prevent server start

- GIVEN `OPENAI_API_KEY` is unset
- WHEN the server starts
- THEN startup succeeds
- AND `synapse_health` and `synapse_stats` work normally
- AND `synapse_backfill` and `synapse_search` return a clear error: `"OPENAI_API_KEY is not set"`

### Requirement: Tool Registration Completeness

All five MCP tools MUST be registered before the server accepts its first
request. Tool registration MUST be idempotent and panic-free.

#### Scenario: All tools registered before first request

- GIVEN a freshly started server
- WHEN a client queries the tool list
- THEN exactly five tools are returned with the names:
  `synapse_health`, `synapse_stats`, `synapse_backfill`, `synapse_search`, `synapse_sync`

### Requirement: Test Isolation

Unit and integration tests MUST NOT require a real `engram.db` or a real
`OPENAI_API_KEY`. All DB-touching tests MUST use `t.TempDir()` with
in-memory or temp-file SQLite instances. All OpenAI calls MUST be
exercised via the `Embedder` mock interface.

#### Scenario: Server wires with mock dependencies in tests

- GIVEN a test uses `t.TempDir()` for `SYNAPSE_DB_PATH` and a stub for `ENGRAM_DB_PATH`
- AND the `Embedder` port is replaced with a `NoopEmbedder`
- WHEN the server is constructed and all tools are called
- THEN no real network calls are made and no real DB files are touched
