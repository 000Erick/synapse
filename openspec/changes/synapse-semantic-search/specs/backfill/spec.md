# backfill Specification

## Purpose

`synapse_backfill` MCP tool: one-time idempotent batch-embedding of all
live Engram observations into `synapse.db`. Reports embedded/skipped/failed counts.

## Requirements

### Requirement: Idempotent Backfill

`synapse_backfill` MUST read all live observations from `engram.db`
(`deleted_at IS NULL`) and embed only those not already present in
`synapse.db`. Re-running backfill on a fully embedded corpus MUST
insert 0 new vectors (skip all). A partial run MUST only embed the
missing observations.

#### Scenario: First backfill embeds all live observations

- GIVEN `engram.db` has 10 live observations and `synapse.db` has 0 vectors
- AND a `MockEmbedder` returning 3072-dim zero vectors
- WHEN `synapse_backfill` is called
- THEN 10 vectors are stored in `synapse.db`
- AND the tool returns `{"embedded": 10, "skipped": 0, "failed": 0}`

#### Scenario: Re-run embeds nothing when all already present

- GIVEN `synapse.db` already has vectors for all 10 live observations
- WHEN `synapse_backfill` is called again
- THEN 0 new vectors are inserted
- AND the tool returns `{"embedded": 0, "skipped": 10, "failed": 0}`

#### Scenario: Partial run embeds only missing observations

- GIVEN `engram.db` has 10 live observations and `synapse.db` has vectors for ids 1–7 only
- WHEN `synapse_backfill` is called
- THEN exactly 3 new vectors are inserted (for ids 8, 9, 10)
- AND the tool returns `{"embedded": 3, "skipped": 7, "failed": 0}`

### Requirement: Batched Embedding Calls

Backfill MUST group observations into batches of at most 100 and call
the `Embedder` once per batch. For ~1040 observations this results in
~11 batches.

#### Scenario: 250 observations produce 3 embedder calls (batches of 100, 100, 50)

- GIVEN `engram.db` has 250 live observations and `synapse.db` is empty
- AND a `MockEmbedder` that records call count
- WHEN `synapse_backfill` is called
- THEN the embedder is called exactly 3 times with sizes [100, 100, 50]

### Requirement: Engram Read-Only During Backfill

Backfill MUST read `engram.db` exclusively via the `EngramReader` port
(read-only). It MUST NOT write to `engram.db` at any point.

#### Scenario: engram.db is not modified during backfill

- GIVEN a temp `engram.db` opened with `mode=ro`
- WHEN `synapse_backfill` runs
- THEN the `engram.db` file modification time is unchanged
- AND no write SQL statement is executed on the engram connection

### Requirement: Failure Reporting

If embedding a batch fails (e.g. mock returns error), backfill MUST
continue processing remaining batches and report the failure count in
the response. It MUST NOT silently drop errors.

#### Scenario: One batch fails — others complete, failure reported

- GIVEN 3 batches of 10, 10, 10 observations
- AND the `MockEmbedder` returns an error on the second batch only
- WHEN `synapse_backfill` is called
- THEN `{"embedded": 20, "skipped": 0, "failed": 10}` is returned
- AND the 20 successfully embedded vectors are persisted

### Requirement: Missing API Key

If `OPENAI_API_KEY` is not set, `synapse_backfill` MUST return an error
immediately without reading any observations.

#### Scenario: Backfill called without API key

- GIVEN `OPENAI_API_KEY` is unset
- WHEN `synapse_backfill` is called
- THEN it returns an error: `"OPENAI_API_KEY is not set"`
- AND no observations are read from `engram.db`
