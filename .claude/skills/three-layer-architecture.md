---
name: three-layer-architecture
description: Use when the user asks to add a new feature, endpoint, domain entity, or CRUD operation to this Go backend. Guides implementation following the project's three-layer architecture (Handler → Service → Repository). Also use when the user asks how a feature is structured or where to put new code.
---

# Three-Layer Architecture Guide

This project follows a strict three-layer architecture. Every feature that touches an HTTP endpoint and the database must be implemented across all three layers in the correct order.

## The Three Layers

```
HTTP Request
     │
     ▼
┌─────────────┐
│   Handler   │  internal/handlers/      — HTTP concerns only
└──────┬──────┘
       │ calls (via interface)
       ▼
┌─────────────┐
│   Service   │  internal/services/      — Business logic
└──────┬──────┘
       │ calls (via interface)
       ▼
┌─────────────┐
│ Repository  │  internal/repositories/  — Data access only
└─────────────┘
       │
       ▼
  Database (SQLC)
```

Each layer depends only on the layer below it, and always through an **interface** — never a concrete struct.

---

## Layer Responsibilities

### Handler (`internal/handlers/<domain>_handler.go`)

- Parse the HTTP request (JSON body, URL params, headers)
- Validate request shape (required fields, formats)
- Call the service
- Map service errors to HTTP status codes using `errors.Is()`
- Write the JSON response via `webutils.WriteJSON()`

**Must NOT contain:** business logic, database calls, password hashing, email sending.

```go
type AccountHandler struct {
    accountService services.AccountService
}

func NewAccountHandler(accountService services.AccountService) *AccountHandler {
    return &AccountHandler{accountService: accountService}
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req createAccountRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        webutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
        return
    }
    if req.FirstName == "" {
        webutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "first name is required"})
        return
    }

    created, err := h.accountService.CreateAccount(r.Context(), models.Account{
        FirstName: req.FirstName,
    }, req.Password)
    if errors.Is(err, services.ErrAlreadyExists) {
        webutils.WriteJSON(w, http.StatusConflict, map[string]string{"error": "account already exists"})
        return
    }
    if err != nil {
        webutils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
        return
    }

    webutils.WriteJSON(w, http.StatusCreated, created)
}
```

### Service (`internal/services/<domain>_service.go`)

- Define the **public interface** for this domain (e.g. `AccountService`)
- Implement business rules: validation, defaults, hashing, orchestration across repos
- Define **sentinel errors** (`var ErrXxx = errors.New(...)`) for conditions handlers need to distinguish
- Wrap errors with context: `fmt.Errorf("creating account: %w", err)`
- May inject multiple repository interfaces to coordinate cross-entity operations

**Must NOT contain:** HTTP status codes, JSON marshaling, raw SQL.

Repository interfaces consumed by services are defined in `internal/services/repositories.go` — not in the repository package. Services own the contract.

```go
// In services/repositories.go
type AccountRepository interface {
    CreateAccount(ctx context.Context, account models.Account) (models.Account, error)
    GetAccount(ctx context.Context, id uuid.UUID) (models.Account, error)
}

// In services/account_service.go
var ErrAlreadyExists = errors.New("account already exists")

type AccountService interface {
    CreateAccount(ctx context.Context, account models.Account, password string) (models.Account, error)
}

type accountService struct {
    accountRepo AccountRepository
}

func NewAccountService(accountRepo AccountRepository) AccountService {
    return &accountService{accountRepo: accountRepo}
}

func (s *accountService) CreateAccount(ctx context.Context, account models.Account, password string) (models.Account, error) {
    hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return models.Account{}, fmt.Errorf("hashing password: %w", err)
    }
    account.PasswordHash = string(hashed)
    if account.SystemRole == "" {
        account.SystemRole = models.SystemRoleUser
    }
    return s.accountRepo.CreateAccount(ctx, account)
}
```

### Repository (`internal/repositories/<domain>_repository.go`)

- Implement the repository interface defined in `services/repositories.go`
- Wrap the SQLC-generated `*database.Queries`
- Map **in**: `models.X` → `database.CreateXParams`
- Map **out**: `database.X` → `models.X` via a private `toModelX()` helper
- No business logic. No error wrapping beyond forwarding.

```go
type accountRepository struct {
    queries *database.Queries
}

func NewAccountRepository(queries *database.Queries) services.AccountRepository {
    return &accountRepository{queries: queries}
}

func (r *accountRepository) CreateAccount(ctx context.Context, account models.Account) (models.Account, error) {
    dbAccount, err := r.queries.CreateAccount(ctx, database.CreateAccountParams{
        FirstName:    account.FirstName,
        PasswordHash: account.PasswordHash,
        SystemRole:   database.UserRole(account.SystemRole),
    })
    if err != nil {
        return models.Account{}, err
    }
    return toModelAccount(dbAccount), nil
}

func toModelAccount(a database.Account) models.Account {
    return models.Account{
        ID:           a.ID,
        FirstName:    a.FirstName,
        PasswordHash: a.PasswordHash,
        SystemRole:   models.SystemRole(a.SystemRole),
    }
}
```

---

## Wiring (`cmd/api/main.go`)

All dependency injection happens in `main()`. The order is always bottom-up:

```
queries (SQLC)
  → repositories (inject queries)
    → services (inject repositories)
      → handlers (inject services)
        → routes (register handlers)
```

When adding a new domain, register its constructor calls in `main.go` in this order. Never create dependencies in the struct itself — always pass them in.

---

## Naming Conventions

| What | Convention | Example |
|---|---|---|
| Handler struct | `<Domain>Handler` | `AccountHandler` |
| Handler file | `<domain>_handler.go` | `account_handler.go` |
| Service interface | `<Domain>Service` | `AccountService` |
| Service struct (private) | `<domain>Service` | `accountService` |
| Service file | `<domain>_service.go` | `account_service.go` |
| Repository interface | `<Domain>Repository` | `AccountRepository` |
| Repository struct (private) | `<domain>Repository` | `accountRepository` |
| Repository file | `<domain>_repository.go` | `account_repository.go` |
| Constructor return type | interface, not struct | `func New...() AccountService` |
| Sentinel errors | `Err<Condition>` in services | `ErrAlreadyExists` |
| DB→model mapper | `toModel<Domain>()` (private) | `toModelAccount()` |

---

## Checklist for Adding a New Feature

When implementing a new domain or endpoint, work bottom-up:

1. **Model** — add/update structs in `internal/models/`
2. **SQL** — write the query in `db/queries/`, run `sqlc generate`
3. **Repository interface** — add method signature to `services/repositories.go`
4. **Repository implementation** — implement in `repositories/<domain>_repository.go`, map types
5. **Service interface** — add method to the `<Domain>Service` interface
6. **Service implementation** — add business logic, define sentinel errors if needed
7. **Handler** — parse request, call service, map errors to status codes
8. **Wiring** — wire in `cmd/api/main.go` if new structs were created
9. **Route** — register the handler method on the router

---

## Error Flow

Errors travel upward and are wrapped at each layer:

```
Repository:  return models.Account{}, err                          // raw DB error
Service:     return ..., fmt.Errorf("creating account: %w", err)  // add context
Handler:     errors.Is(err, services.ErrXxx) → HTTP status code   // classify
```

Define a sentinel error in the service whenever a handler needs to distinguish a specific condition (e.g. not found vs. conflict vs. forbidden). Use `errors.Is()` in the handler — never string-match error messages.

---

## Applying This Pattern in a Microservice Architecture

This three-layer pattern applies within each individual microservice unchanged. What changes is what each layer talks to:

- **Handler layer** becomes the service boundary. Instead of only receiving browser HTTP requests, it may also handle gRPC calls, message-queue events, or REST calls from other services.
- **Repository layer** may back onto a local database, a cache, or a downstream service called via HTTP/gRPC client. The interface abstraction stays the same — the service layer is unaware whether data is local or remote. Model a remote dependency as a "client repository" that implements the same `<Domain>Repository` interface.
- **Service layer** is unchanged. It never knows whether its repositories are backed by Postgres or a remote API.

Cross-cutting concerns (propagating auth tokens, distributed tracing, circuit breakers) belong in middleware at the handler layer, not inside service or repository code.

---

## Key Rules (Summary)

- Handlers never touch the database. Repositories never contain business logic.
- Every inter-layer dependency is an interface, never a concrete struct.
- Constructors always return the interface type, not the struct.
- Repository interfaces live in the `services` package — services own the data contract.
- Sentinel errors are defined in `services`, checked in handlers with `errors.Is()`.
- Mapping between DB types and domain models stays inside the repository.
- Dependency wiring is centralised in `cmd/api/main.go`.
