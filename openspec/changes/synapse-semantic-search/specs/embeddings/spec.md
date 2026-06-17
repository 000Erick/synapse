# embeddings Specification

## Purpose

OpenAI `text-embedding-3-large` HTTP client behind the `Embedder` port,
with batching support and a mock implementation for strict-TDD tests.

## Requirements

### Requirement: Embedder Port Contract

The system MUST define an `Embedder` interface with a single method that
accepts a batch of strings and returns a batch of `float32` vectors (one
per input). The interface MUST be the only embedding dependency visible
to domain and use-case layers.

#### Scenario: Port is the sole embedding dependency in domain/use-case layers

- GIVEN the domain and ports packages
- WHEN their imports are inspected
- THEN no import of `"net/http"`, `"github.com/ediazs/synapse/internal/adapter/openai"`, or any concrete embedder appears

### Requirement: OpenAI Embedder — Batch Embedding

The `OpenAIEmbedder` MUST call the OpenAI embeddings endpoint
(`POST /v1/embeddings`) with `model=text-embedding-3-large`.
Each batch MUST contain at most 100 inputs. The response MUST be
validated: each returned embedding MUST have exactly 3072 dimensions.

#### Scenario: Single batch of 3 texts returns 3 vectors of 3072 dims

- GIVEN a mocked HTTP server returning valid 3072-dim embeddings
- WHEN `Embed(["a", "b", "c"])` is called
- THEN 3 vectors are returned, each with length 3072
- AND exactly 1 HTTP call was made to the embeddings endpoint

#### Scenario: 101 texts are split into 2 HTTP calls

- GIVEN a mocked HTTP server and 101 input strings
- WHEN `Embed` is called
- THEN 2 HTTP calls are made (one for 100 inputs, one for 1)
- AND 101 vectors are returned in order

#### Scenario: API returns wrong dimension — error is propagated

- GIVEN a mocked HTTP server that returns 1536-dim vectors
- WHEN `Embed` is called
- THEN an error is returned: dimension mismatch `"expected 3072, got 1536"`

### Requirement: Missing API Key Error

If `OPENAI_API_KEY` is empty at the time `Embed` is called, the embedder
MUST return a clear error without making any HTTP call.

#### Scenario: Embed called without API key

- GIVEN `OPENAI_API_KEY` is empty
- WHEN `Embed(["text"])` is called on the OpenAI embedder
- THEN no HTTP request is made
- AND the error message contains `"OPENAI_API_KEY is not set"`

### Requirement: Mock Embedder for Tests

A `MockEmbedder` (or `NoopEmbedder`) MUST be provided that returns
deterministic, configurable vectors without any network call. It MUST
implement the same `Embedder` interface. Tests MUST use this mock; no
test in the suite MUST make real OpenAI HTTP calls.

#### Scenario: MockEmbedder returns configured vectors

- GIVEN a `MockEmbedder` configured to return vector `[0.1, 0.2, ..., 3072 dims]`
- WHEN `Embed(["any text"])` is called
- THEN the configured vector is returned
- AND no HTTP call is made

#### Scenario: Full test suite passes without real OpenAI key

- GIVEN `OPENAI_API_KEY` is unset
- WHEN `go test ./...` is run
- THEN all embedding tests pass using the `MockEmbedder`

### Requirement: HTTP Timeout and Error Handling

The `OpenAIEmbedder` MUST apply a configurable HTTP timeout (default 30s).
On non-2xx HTTP responses, it MUST return an error including the status
code and body excerpt.

#### Scenario: HTTP 429 rate-limit returns descriptive error

- GIVEN a mocked HTTP server returns status 429 with body `{"error":"rate limit"}`
- WHEN `Embed` is called
- THEN an error is returned containing `"429"` and the response body excerpt
