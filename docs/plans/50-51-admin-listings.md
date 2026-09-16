# Implementation plan — #50 Admin: List farms, #51 Admin: List accounts

Both issues are title-only, so the scope below is inferred from the surrounding
code and the neighbouring issues (#47 public farm details, #49 verify farms,
#40 revenue statistics). Assumptions are stated explicitly in §1 — correct them
there and the rest of the plan follows.

The guiding constraint: **these two endpoints are read-only aggregate views for
an admin back-office, so they are built the way `GET /api/statistics` is built**,
not the way `GET /api/fields` is. §2 says what that means concretely.

---

## 1. Assumptions

1. **A "farm" is a `farmer` subtype row plus its `account`.** There is no `farm`
   table; `farmer.farm_name` / `farmer.postal_code` (migration
   `20260909071744_accounts.sql`) are the whole of it, and a farmer account owns
   exactly one farm. "List farms" therefore means "list farmer accounts, farm
   side first".
2. **Both endpoints are admin-only**, read-only, and back a back-office screen —
   an operator wants to scan the platform, spot farms with no fields, and find
   an account by email. They are not a public directory (#47 covers that) and
   they do not mutate anything (#49 covers verification).
3. **Verification is out of scope** and lands with #49. The plan leaves a seam
   for it (§7) but adds no column and no field.
4. The frontend has no admin screen yet, so `openapi.yml` is the only contract
   that has to move in this change.

If (1) is wrong — i.e. a farmer should be able to own several farms — this
becomes a schema change and a much larger issue. Worth confirming before step 2
of either issue; nothing else in the plan depends on it.

---

## 2. Alignment with `GET /api/statistics`

`ARCHITECTURE.md` §9 and `sql/queries/statistics.sql` establish five rules. All
five carry over, and carrying them over is most of the design work:

| Statistics rule | How it applies here |
| --- | --- |
| **Scope comes from the caller's role, never a parameter.** No `?scope=`, no farmer id in the request. | Both routes are admin-only and always platform-wide. There is deliberately **no `?farmerId=`** on the farms list — an admin needs no such filter, and adding one creates a tampering surface for the day a second role reaches these routes. Filtering is by farm name / postal code / email, never by identity. |
| **One query per response, one round trip, one MVCC snapshot** — so `plots.rented` and `rentals.active` can never disagree within a response. | The row count for pagination is `(COUNT(*) OVER ())` inside the same `SELECT`, not a second `COUNT` query. `total` and `items` therefore come from the same snapshot; a registration between two round trips cannot make page 2 skip a row. |
| **Cast every column (`::bigint`, `::float8`) and `COALESCE` every aggregate**, because sqlc has no PostGIS catalog and `SUM` over an empty set is `NULL`. | Same discipline verbatim. Sharper here: a farm with **zero fields** is the common case on a fresh platform, so every per-farm aggregate is `COALESCE(..., 0)` and the join to the aggregates is a `LEFT JOIN LATERAL` — a farm with nothing must still appear as a row of zeros, exactly as a farmer with nothing gets zeros from `GetFarmStatistics` rather than no row. |
| **Derived figures are computed in Go, not SQL** (`withDerivedStatistics`), so divide-by-zero is a guarded branch rather than a `NULL` crossing into pgx. | `available` and `occupancyRate` per farm are derived in the service by the *same code*: extract the body of `withDerivedStatistics` into `derivePlotFigures(models.PlotStatistics) models.PlotStatistics` and call it from both services. One definition of occupancy on the platform. |
| **The same "rented right now" predicate everywhere**: `r.period @> CURRENT_TIMESTAMP`, so an expired rental stops counting with no cleanup job. | Reused verbatim. This is what makes the two endpoints *reconcile*: `SUM(fieldCount)` over every row of `/api/admin/farms` equals `fields.total` from platform statistics, and likewise for plots and rented plots. An admin comparing the two screens must not see different numbers. Worth an explicit integration-test assertion (§6). |

Two places where the statistics pattern deliberately **does not** carry over:

- `statistics.sql` warns "keep every cross-joined CTE free of `GROUP BY` — a CTE
  returning no rows would empty the whole result". That warning is about a
  single-row cross join. These are per-farm rows, so aggregation *is* grouped;
  the equivalent hazard is the inner join, which is why the aggregates hang off
  `LEFT JOIN LATERAL`.
- Statistics returns a flat object. A list needs an envelope to carry `total`
  (§3).

---

## 3. Shared foundation (build once, both issues use it)

### 3a. Route group `/api/admin`

```go
r.Route("/api/admin", func(r chi.Router) {
    r.Use(appmiddleware.RequireAuth(authService))
    r.Use(appmiddleware.RequireRole(models.RoleAdmin))

    r.Get("/farms", farmHandler.ListFarms)       // #50
    r.Get("/accounts", accountHandler.ListAccounts) // #51
})
```

Why a dedicated prefix rather than `GET /api/farms` and `GET /api/accounts`:

- `ARCHITECTURE.md` §13 gap 3 records that admins have no route of their own
  besides statistics. This closes it, and the gap entry should be updated in the
  same change.
- #47 wants a *public* farm view. Keeping the admin listing at
  `/api/admin/farms` leaves `/api/farms/{id}` free for it, with a different
  response shape and no auth, instead of forcing one path to serve two
  audiences.

### 3b. Pagination

`ARCHITECTURE.md` §13 gap 9 records that nothing is paginated. An admin list of
every account is exactly where that stops being acceptable, so pagination is in
scope for these two endpoints (and only these two — retrofitting the existing
list routes is a separate issue).

- Query params `limit` (default 20, max 100) and `offset` (default 0, min 0).
- `limit` reuses the shape already in `plot_search_handler.go`
  (`parseNearestPlotsLimit`, default 20 / max 100). Promote it to one shared
  helper in `handlers` — `parseLimit(raw string, def, max int32)` — rather than
  writing a third copy.
- Response envelope, because `total` has nowhere else to live:

  ```json
  { "items": [ ... ], "total": 137, "limit": 20, "offset": 0 }
  ```

  This differs from `GET /api/fields` / `/api/rentals` / `/api/announcements`,
  which return bare arrays. That is intentional and should be called out in the
  PR: bare arrays cannot carry a count, and the envelope matches the object
  shape `/api/statistics` already returns. New paginated routes use the
  envelope; the existing bare-array routes are not changed here.
- `ORDER BY` always ends with `, a.id` as a tiebreaker. Without it, two farms
  with the same name can swap places between page 1 and page 2 and a row is
  silently skipped.
- Note for the PR description: `(COUNT(*) OVER ())` yields no row — and hence
  `total: 0` — when the requested page is past the end. Acceptable (the client
  already knows the total from the previous page); documented in `openapi.yml`
  rather than worked around with a second query.

### 3c. Model reuse

`models.FieldStatistics` and `models.PlotStatistics` already carry exactly the
per-farm figures #50 needs, with the JSON names the frontend already renders on
the statistics dashboard. Reuse them — do not define a parallel
`FarmPlotCounts`. That keeps `areaSquareMeters` / `occupancyRate` meaning one
thing platform-wide.

```go
// internal/models/page.go
type Page[T any] struct {
    Items  []T
    Total  int64
    Limit  int32
    Offset int32
}
```

---

## 4. #50 — Admin: List farms

`GET /api/admin/farms`

**Query params:** `limit`, `offset`, `q` (case-insensitive substring over farm
name, owner name and owner email), `postalCode` (exact).

**Response item:**

```json
{
  "accountId": "…", "farmName": "Green Acres", "postalCode": 76133,
  "owner": { "firstName": "Old", "lastName": "MacDonald", "email": "…@example.com" },
  "createdAt": "2026-09-13T16:21:04Z",
  "fields": { "total": 3, "areaSquareMeters": 128394.5 },
  "plots":  { "total": 42, "rented": 17, "available": 25,
              "areaSquareMeters": 91002.1, "occupancyRate": 0.4047619 },
  "activeRentals": 17
}
```

`fields` and `plots` are byte-for-byte the shapes `/api/statistics` returns, so
the frontend reuses its existing renderers.

Steps, bottom-up per the skill's checklist and `ARCHITECTURE.md` §14:

1. **Model** — `internal/models/farm.go`: `FarmListing` (fields above, embedding
   `FieldStatistics` / `PlotStatistics`), plus `FarmListFilter{Query string;
   PostalCode *int32; Limit, Offset int32}`. Add `internal/models/page.go`.
2. **SQL** — `sql/queries/farm.sql`, `-- name: ListFarms :many`:
   - `FROM farmer f JOIN account a ON a.id = f.account_id`
   - `LEFT JOIN LATERAL (SELECT COUNT(*)::bigint AS total, COALESCE(SUM(ST_Area(coordinates::geography)::float8), 0)::float8 AS area FROM field WHERE farmer = f.account_id) fields ON TRUE`
   - a matching lateral for plots, counting `rented` with
     `COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM rental r WHERE r.plot = p.id AND r.period @> CURRENT_TIMESTAMP))` — the identical predicate to `statistics.sql`
   - a lateral for `active_rentals` over `rental`→`plot`→`field`
   - `WHERE (sqlc.narg('query')::text IS NULL OR f.farm_name ILIKE '%' || sqlc.narg('query')::text || '%' OR a.email ILIKE … OR a.first_name ILIKE … OR a.last_name ILIKE …)` and the same `narg` pattern for `postal_code`
   - `(COUNT(*) OVER ())::bigint AS total_count`
   - `ORDER BY f.farm_name, a.id LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off')`
   - run `sqlc generate`.
3. **Repository interface** — `FarmRepository.ListFarms(ctx, models.FarmListFilter) (models.Page[models.FarmListing], error)` in `internal/services/repositories.go`.
4. **Repository** — `internal/repositories/farm_repository.go`: map filter →
   `database.ListFarmsParams`, rows → `[]models.FarmListing` via a private
   `toModelFarmListing`, and lift `total_count` off the first row (zero rows →
   `Total: 0`). No business logic.
5. **Service interface** — `internal/services/farm_service.go`: `FarmService`
   with `ListFarms(ctx, role models.Role, filter models.FarmListFilter) (models.Page[models.FarmListing], error)`.
6. **Service** — mirror `statisticsService.GetStatistics`: `switch role`, admin
   proceeds, everything else returns `ErrForbidden` as defence in depth behind
   `RequireRole(admin)`, with the same comment explaining why the check is there
   twice. Clamp `Limit`/`Offset` to sane values, then call
   `derivePlotFigures` (§2) on each row to fill `available` / `occupancyRate`.
7. **Handler** — `internal/handlers/farm_handler.go`: parse the four params with
   the shared `parseLimit`, 400 on a malformed one with the same wording style as
   `plot_search_handler.go`, `errors.Is(err, services.ErrForbidden)` → 403,
   anything else → `slog.Error` + 500, success → `webutils.WriteJSON` of the
   envelope.
8. **Wiring** — `cmd/api/main.go`, bottom-up in the existing block:
   `farmRepo` → `farmService` → `farmHandler`.
9. **Route** — the `/api/admin` group in `routes.go` (§3a); `NewRouter` gains
   the handler parameter.
10. **OpenAPI** — `/api/admin/farms` plus `FarmListing`, `FarmOwner` and
    `FarmPage` schemas, reusing the existing `FieldStatistics` /
    `PlotStatistics` `$ref`s.

**Ordering note:** default is farm name, which is not unique — hence the `a.id`
tiebreaker in §3b. `ORDER BY a.created_at DESC` is the better default if the
screen is "who joined recently"; pick one in review, do not make it a
client-supplied `sort` param in this change.

---

## 5. #51 — Admin: List accounts

`GET /api/admin/accounts`

**Query params:** `limit`, `offset`, `role` (`admin` | `farmer` | `customer`),
`q` (substring over email, first name, last name).

**Response item:**

```json
{ "id": "…", "firstName": "Old", "lastName": "MacDonald",
  "email": "…@example.com", "role": "farmer",
  "createdAt": "2026-09-13T16:21:04Z" }
```

Steps:

1. **Model** — extend `internal/models/account.go` with `AccountListing`
   (`Account` minus `PasswordHash`, plus `CreatedAt`) and
   `AccountListFilter{Role *Role; Query string; Limit, Offset int32}`.
   `AccountListing` must **not** carry `PasswordHash` at all — not merely omit it
   from the JSON tag. Do not reuse `models.Account` here.
2. **SQL** — `sql/queries/account.sql`, `-- name: ListAccounts :many`. Derive
   `role` with the same `CASE` over the three subtype `LEFT JOIN`s that
   `GetAccountByEmail` uses — one definition of "role" in the codebase. The
   `CASE` alias is not referencable from `WHERE`, so wrap the projection in a CTE
   and filter the CTE:
   ```sql
   WITH accounts AS (SELECT a.id, …, CASE … END AS role FROM account a LEFT JOIN …)
   SELECT *, (COUNT(*) OVER ())::bigint AS total_count
   FROM accounts
   WHERE (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role')::text)
     AND (sqlc.narg('query')::text IS NULL OR email ILIKE … OR …)
   ORDER BY created_at DESC, id
   LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');
   ```
   Then `sqlc generate`.
3. **Repository interface** — add `ListAccounts(ctx, models.AccountListFilter)
   (models.Page[models.AccountListing], error)` to the existing
   `AccountRepository`.
4. **Repository** — implement in `internal/repositories/account_repository.go`
   next to `GetAccountByEmail`, mapping via `toModelAccountListing`.
5. **Service** — a new `AccountService` in
   `internal/services/account_service.go` (there is none today; `authService`
   talks to `accountRepo` directly). Same admin-or-`ErrForbidden` switch as #50,
   same limit clamping. Validate `role` against the three known values and
   return `ErrInvalidInput` (or a new `ErrInvalidRole` sentinel) for anything
   else — the handler maps it to 400 rather than letting an unknown role return
   a silently empty page.
6. **Handler** — `internal/handlers/account_handler.go`, same error mapping as
   #50 plus 400 for a bad `role`.
7. **Wiring + route + OpenAPI** — as in #50, steps 8–10.

**Edge case worth handling rather than ignoring:** an account with no subtype row
derives the empty role `''` (see the `ELSE ''` in `GetAccountByEmail`, and the
positive-membership test in `GetAllRecipients` that exists to avoid mailing
such rows). Surface it as `"role": null` rather than `""`, so an admin can
actually *see* the orphaned account this endpoint is well placed to expose.
Document it in `openapi.yml` as nullable.

---

## 6. Tests

Mirroring `ARCHITECTURE.md` §11 and the statistics tests:

**Service unit tests** (`*_service_test.go`, fake repo in the file, table-driven —
model on `statistics_service_test.go`'s `fakeStatisticsRepo`):

- admin reaches the repository; farmer and customer get `ErrForbidden` and the
  repository is never called.
- the filter reaches the repository unmodified except for clamping.
- `limit` above the max and a negative `offset` are clamped, not rejected at this
  layer.
- (#50) `available` / `occupancyRate` are derived per row, including a farm with
  `Total: 0` plots — the divide-by-zero guard.
- (#51) an unrecognised `role` filter returns the invalid-input sentinel.

**Repository integration tests** (`*_integration_test.go`, testcontainers,
skipped under `-short`). Reuse `seedFarmWithPlots` and `seedCustomer` from the
existing integration tests rather than writing new seeding:

- (#50) a farmer with no fields appears as a row of zeros — the `LEFT JOIN
  LATERAL` regression test, and the single most likely bug in this change.
- (#50) **reconciliation:** seed two farms, then assert that the summed
  `fieldCount` / `plotCount` / `rentedPlotCount` across `ListFarms` equal
  `GetPlatformStatistics`'s `fields.total` / `plots.total` /
  `plots.rented`. This is the assertion that keeps the two screens honest.
- (#50) a rental whose period has ended does not count toward `rented`.
- (#51) the derived role matches the subtype table for one account of each of
  the three roles; an account with no subtype row comes back with an empty role.
- both: `total` is the unfiltered-by-page count — seed 3, request `limit=2`, get
  2 items and `total: 3`; page 2 returns the third with no overlap.
- both: filters narrow both `items` and `total`.

**Handler-level:** none today (there are no handler tests in the repo); keep it
that way and cover the mapping through the service tests.

---

## 7. Docs to update in the same change

- `openapi.yml` — both paths and their schemas. This is the contract with the
  frontend repo; `ARCHITECTURE.md` §14 makes it part of the change, not a
  follow-up.
- `ARCHITECTURE.md` §5 — two rows in the route table, the `/api/admin` node in
  the mermaid graph.
- `ARCHITECTURE.md` §13 — gap 3 (admins now have their own route group) and gap
  9 (pagination now exists, on these two routes only).
- `ARCHITECTURE.md` — a short subsection next to §9 stating the reconciliation
  invariant: the per-farm figures and the platform statistics come from the same
  predicates and must agree.
- No `AGENT.md` change: no new dependency, no new config, no new convention
  beyond the pagination envelope, which belongs in `ARCHITECTURE.md`.

**Seam for #49 (verify farms):** verification adds a column to `farmer`, a field
on `FarmListing`, a property in `openapi.yml` and — at most — a `?verified=`
filter on this endpoint. It adds no new query, no new service and no new route
to *this* work, provided #50 ships the `FarmService` / `farm.sql` pair rather
than folding the listing into an `AdminService`. That is the main reason for the
domain split in §8.

---

## 8. Decisions taken, with the alternative

- **Domain-named services (`FarmService`, `AccountService`) over one
  `AdminService`.** "Admin" is an audience, not a domain, and the naming table in
  `.claude/skills/three-layer-architecture.md` is written in domains. The split
  also means #47 (public farm details) and #49 (verify farms) extend
  `FarmService` instead of moving code out of an `AdminService` later. Cost:
  `NewRouter` grows two parameters — it already takes ten, and a config struct
  for it is worth doing at some point, but not in this change.
- **Envelope response over bare array** — see §3b.
- **`limit`/`offset` over cursor pagination.** Offsets are fine at this scale and
  match the existing `limit` param on `/api/plots/nearest`. Cursors would be the
  answer at a size this project will not reach.
- **No `sort` query param.** A client-chosen sort column is an injection surface
  for the amount of value it adds here; if the back-office needs it, it is its
  own issue.

## 9. Out of scope

Farm verification (#49), public farm details (#47), revenue figures (#40),
mutating accounts (suspend / delete / role change), CSV export, retrofitting
pagination onto `/api/fields`, `/api/rentals` and `/api/announcements`, and rate
limiting.

## 10. Sequencing

#51 first: it is the smaller of the two and it is what forces the shared pieces
(`models.Page`, `parseLimit`, the `/api/admin` group, the envelope in
`openapi.yml`) into existence. #50 then adds only its own query, model and
handler. Two PRs, `#51 → #50`, or one PR if the reviewer prefers to see the
envelope convention decided once.
