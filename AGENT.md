# AGENT.md

Guidance for AI agents (and humans) working in this repository.

> **Status: active.** The core API is in place — accounts and auth, fields and plots,
> spatial plot search, rentals, statistics, and outbound notifications. Known gaps are catalogued in
> [ARCHITECTURE.md §13](ARCHITECTURE.md#13-known-gaps-and-rough-edges). Sections marked
> **TBD** are decisions that have not been made yet — when you make one, update this
> file in the same change.

## Project

**Bauer as a Service (BaaS)** — the backend (HTTP API) for the BaaS webapp.

- Module: `github.com/Neue-Konzepte-BaaS/backend`
- Repo: <https://github.com/Neue-Konzepte-BaaS/backend>
- This is a university project ("Neue Konzepte").

### Stack

| Layer    | Choice                        |
| -------- | ----------------------------- |
| Backend  | Go (see `go.mod` for version) |
| Database | PostgreSQL + PostGIS          |
| Frontend | React — **separate repo**     |

### Scope boundary

This repo contains **only** the backend. The React frontend lives in its own repository:
<https://github.com/Neue-Konzepte-BaaS/frontend>. Do not add frontend code, build
tooling, or assets here. The contract between the two is the HTTP API — if you change it in a
breaking way, say so explicitly in your summary so the frontend can be updated.

## Layout

The project is using the three-layer-architecture, and has seperate folders like sql/ for sql queries and schemas.

**[ARCHITECTURE.md](ARCHITECTURE.md) is the detailed reference**: layering and the
dependency inversion between `services` and `repositories`, request flows, where each
business rule is enforced, the SQLSTATE-to-HTTP error table, and a step-by-step
checklist for adding a feature (§14). Read it before adding a new endpoint. This file
stays the short version — keep the two consistent.

## Commands

```sh
go run .          # run locally
go build ./...    # build
go test ./...     # run tests
go vet ./...      # static checks
gofmt -l .        # list unformatted files (should print nothing)
```

`make` wraps the common flows: `make db-up` (podman compose), `make up`, `make build`,
and `make migrate` / `migrate-up` / `migrate-down` (dbmate, reading `.env`).

CI is `.github/workflows/ci.yml`: build, lint (`gofmt -l` must be empty, plus
golangci-lint), unit tests (`go vet` and `go test -short -race`), and integration tests.

## Configuration

Config comes from environment variables. `.env` is gitignored and must never be
committed; neither may connection strings, passwords, or API keys — not in code, not in
tests, not in this file.

`.env.example` lists every variable; copy it to `.env` to run locally. `config.Load()`
parses and validates them at startup and fails fast.

| Variable | Default | Notes |
| ----------------- | ------- | ------------------------------------------------------- |
| `DATABASE_URL`    | —       | Required; must parse as a URL with a scheme. |
| `DB_AUTO_MIGRATE` | `false` | Parsed and validated but **not read** — `main.go` migrates unconditionally. |
| `JWT_SECRET`      | —       | Required; at least 32 characters. |
| `COOKIE_SECURE`   | `true`  | Set `false` only for local http development. |
| `SAME_SITE_STRICT`| `true`  | |
| `CORS_ENABLED`    | `false` | |
| `FRONTEND_URL`    | —       | Required when `CORS_ENABLED` is true; must be an absolute URL. |
| `SMTP_ENABLED`    | `false` | Off means notifications are logged, not sent. |
| `SMTP_HOST`       | —       | Required when `SMTP_ENABLED` is true. |
| `SMTP_PORT`       | `587`   | STARTTLS; implicit TLS on 465 is not supported. |
| `SMTP_USERNAME`   | —       | Empty means the sender authenticates with nothing. |
| `SMTP_PASSWORD`   | —       | |
| `SMTP_SENDER_NAME`  | —     | Required when `SMTP_ENABLED` is true. |
| `SMTP_SENDER_EMAIL` | —     | Required when `SMTP_ENABLED` is true; must parse as an address. |

`POSTGRES_USER`, `POSTGRES_PASSWORD` and `POSTGRES_DB` are read by `compose.yml` and the
`Makefile`'s dbmate targets, not by the Go application.

## Conventions

- Return errors, don't panic in request paths. Wrap with context:
  `fmt.Errorf("loading user %d: %w", id, err)`.
- Prefer the standard library. Add a dependency only when it clearly pays for itself,
  and mention new dependencies in your change summary.
- Keep packages small and named after what they provide, not what they contain
  (`user`, not `models`/`utils`).
- Comments explain *why*, not *what*. Match the density of the surrounding code.

## Notifications

Outbound mail goes through `services.NotificationService`, backed by an
`EmailSender` (`internal/repositories/email_sender.go`) that is SMTP in production
and a console logger when `SMTP_ENABLED` is false. Templates are `html/template`
files embedded from `internal/emailtemplates` — do not read them from disk, the
runtime image ships only the binary and the migrations.

Fan-out runs on `services.Dispatcher` after the response is written, so delivery
is best-effort: nothing is retried, and failures are logged rather than returned.
`main.go` shuts down gracefully so those sends are not killed mid-flight.

## Database

PostgreSQL with the **PostGIS** and **btree_gist** extensions — both are required, not
optional: geometry columns and the rental no-overlap exclusion constraint depend on
them. We use sqlc to query the database and dbmate for migrations. A local db for
testing can be started via the compose.yml file.

## API

chi is the http router.

Document all API routes in the projects openapi.yml

## Testing

Standard `go test`. Tests live next to the code they cover, as `*_test.go`. Prefer
table-driven tests.

For integration tests, we use the library testcontainers to spin up postgres instances.
They live in `*_integration_test.go` and skip under `go test -short`, which is how CI
separates the unit and integration jobs — there is no build tag.

## Working in this repo

- Small, focused commits with imperative messages ("add user endpoint").
- If tests fail or something is left unfinished, say so plainly instead of glossing over it.
- Ask before adding or changing infrastructure (CI, containers, deployment) — that is a
  project-wide decision, not an implementation detail. CI and a `Containerfile` now
  exist; extending them still counts.
- Keep this file current: it is the shared context for everyone working here.
