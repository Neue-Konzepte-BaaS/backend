# AGENT.md

Guidance for AI agents (and humans) working in this repository.

> **Status: early / greenfield.** Right now this repo is little more than a Go module
> and a `main.go`. Sections marked **TBD** are decisions that have not been made yet —
> when you make one, update this file in the same change.

## Project

**Bauer as a Service (BaaS)** — the backend (HTTP API) for the BaaS webapp.

- Module: `github.com/Neue-Konzepte-BaaS/backend`
- Repo: <https://github.com/Neue-Konzepte-BaaS/backend>
- This is a university project ("Neue Konzepte").

### Stack

| Layer    | Choice                        |
| -------- | ----------------------------- |
| Backend  | Go (see `go.mod` for version) |
| Database | PostgreSQL                    |
| Frontend | React — **separate repo**     |

### Scope boundary

This repo contains **only** the backend. The React frontend lives in its own repository
in the `Neue-Konzepte-BaaS` GitHub org. Do not add frontend code, build tooling, or
assets here. The contract between the two is the HTTP API — if you change it in a
breaking way, say so explicitly in your summary so the frontend can be updated.

## Layout

```
.
├── main.go       # entrypoint
├── go.mod
└── AGENT.md
```

Planned structure (adopt as the code grows, don't create empty dirs up front):

- `cmd/` — entrypoints, if more than one binary is ever needed
- `internal/` — application code; not importable from outside the module
- `migrations/` — SQL schema migrations, checked in and ordered

## Commands

```sh
go run .          # run locally
go build ./...    # build
go test ./...     # run tests
go vet ./...      # static checks
gofmt -l .        # list unformatted files (should print nothing)
```

There is no Makefile, task runner, or CI pipeline yet — **TBD**.

## Configuration

Config comes from environment variables. `.env` is gitignored and must never be
committed; neither may connection strings, passwords, or API keys — not in code, not in
tests, not in this file.

There is no `.env.example` yet. When the first env var is introduced, create one and
document the variables in a table here.

## Conventions

- Standard Go style: `gofmt`, and follow [Effective Go](https://go.dev/doc/effective_go).
  Keep the code idiomatic rather than importing patterns from other languages.
- Return errors, don't panic in request paths. Wrap with context:
  `fmt.Errorf("loading user %d: %w", id, err)`.
- Prefer the standard library. Add a dependency only when it clearly pays for itself,
  and mention new dependencies in your change summary.
- Keep packages small and named after what they provide, not what they contain
  (`user`, not `models`/`utils`).
- Comments explain *why*, not *what*. Match the density of the surrounding code.

## Database

PostgreSQL. **TBD:** driver/access layer (`database/sql` + `pgx`, `sqlc`, an ORM, …),
migration tool, and local setup (Docker Compose vs. a local server).

Once decided, document here: how to start a local DB, how to run and create migrations,
and how tests get a database.

Until then: schema changes belong in checked-in migration files, never applied by hand
to a shared database.

## API

**TBD:** router/framework, URL and versioning scheme (e.g. `/api/v1/...`), auth,
request/response and error JSON shapes.

Whatever is chosen, keep it consistent across endpoints and document it here — this
section is what the frontend developers will read.

## Testing

Standard `go test`. Tests live next to the code they cover, as `*_test.go`. Prefer
table-driven tests.

**TBD:** integration test strategy against a real Postgres.

## Working in this repo

- Small, focused commits with imperative messages ("add user endpoint").
- Run `go build ./...`, `go test ./...` and `gofmt -l .` before you call a change done.
- If tests fail or something is left unfinished, say so plainly instead of glossing over it.
- Ask before adding infrastructure (CI, Docker, deployment) — that is a project-wide
  decision, not an implementation detail.
- Keep this file current: it is the shared context for everyone working here.
