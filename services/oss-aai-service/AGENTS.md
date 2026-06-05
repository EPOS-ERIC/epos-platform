# AGENTS

## Service Conventions

- Keep authentication and token issuance behavior deterministic and auditable.
- Do not log credentials, tokens, client secrets, or password hashes.
- Keep persisted SQLite data under `data/` and out of source control.
