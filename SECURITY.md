# Security Policy

## Reporting a vulnerability

Please **do not open a public issue** for security vulnerabilities.

Instead, use GitHub's private vulnerability reporting:

1. Go to the **Security** tab of this repository.
2. Click **Report a vulnerability**.
3. Describe the issue and, if possible, steps to reproduce.

You'll get a response as soon as possible. Once a fix is available and released, the report can be disclosed publicly.

## Handling of secrets

Synapse needs an `OPENAI_API_KEY` to generate embeddings. A few guarantees about how it is handled:

- The key is read **only** from the environment (or a local `.env` that is **git-ignored** and never committed).
- The key is **never logged**, never written to `synapse.db`, and never sent anywhere except OpenAI's embeddings endpoint over HTTPS.
- `engram.db` is opened **read-only** and is never modified.

If you find any code path that logs, persists, or transmits the key anywhere else, please report it using the process above — that is considered a security bug.

## Supported versions

This project is pre-1.0. Security fixes land on the latest released version.
