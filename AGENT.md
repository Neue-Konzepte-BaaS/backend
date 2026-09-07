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

This repo contains **only** the backend. The React frontend lives in its own repository:
<https://github.com/Neue-Konzepte-BaaS/frontend>. Do not add frontend code, build
tooling, or assets here. The contract between the two is the HTTP API — if you change it in a
breaking way, say so explicitly in your summary so the frontend can be updated.

## Layout

The project is using the three-layer-architecture, and has seperate folders like sql/ for sql queries and schemas.

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

- Return errors, don't panic in request paths. Wrap with context:
  `fmt.Errorf("loading user %d: %w", id, err)`.
- Prefer the standard library. Add a dependency only when it clearly pays for itself,
  and mention new dependencies in your change summary.
- Keep packages small and named after what they provide, not what they contain
  (`user`, not `models`/`utils`).
- Comments explain *why*, not *what*. Match the density of the surrounding code.

## Database

PostgreSQL is used. We use sqlc to query the database and dbmate for migrations. A local db for testing can be started via the compose.yml file.

## API

We will use chi as http router in this project.

## Testing

Standard `go test`. Tests live next to the code they cover, as `*_test.go`. Prefer
table-driven tests.

For integration tests, we use the library testcontainers to spin up postgres instances.

## Working in this repo

- Small, focused commits with imperative messages ("add user endpoint").
- If tests fail or something is left unfinished, say so plainly instead of glossing over it.
- Ask before adding infrastructure (CI, Docker, deployment) — that is a project-wide
  decision, not an implementation detail.
- Keep this file current: it is the shared context for everyone working here.
