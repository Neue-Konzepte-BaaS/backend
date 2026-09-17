# Graph Report - backend  (2026-09-17)

## Corpus Check
- 109 files · ~159,239 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 5 file(s) not represented in the graph (top: (none) 4, .example 1)

## Summary
- 832 nodes · 2228 edges · 43 communities (29 shown, 1 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 149 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `bf809796`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- net/http.ResponseWriter
- NotificationService
- AnnouncementWithFarm
- AuthService
- main
- testing.T
- Statistics
- github.com/google/uuid.UUID
- context.Context
- field
- github.com/twpayne/go-geom.Polygon
- FarmListing
- NewRouter
- Schwarzes Brett (Farmer Announcement Board)
- Three-Layer Architecture Pattern
- Config
- OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)
- crop
- email_sender_test.go
- .ListAccounts
- account.sql.go
- Service Layer (internal/services)
- plot.sql.go
- GET/POST /api/rentals
- ListFarmsRow
- GET /api/statistics Role-derived Scope
- github.com/jackc/pgx/v5/pgtype.Timestamptz
- Login Flow (argon2id + JWT)
- announcement.sql.go
- github.com/Neue-Konzepte-BaaS/backend

## God Nodes (most connected - your core abstractions)
1. `main()` - 39 edges
2. `Account` - 29 edges
3. `New()` - 26 edges
4. `WriteError()` - 25 edges
5. `WriteJSON()` - 23 edges
6. `Queries` - 22 edges
7. `Role` - 21 edges
8. `MustClaimsFromContext()` - 20 edges
9. `setupTestDB()` - 20 edges
10. `NotificationService` - 20 edges

## Surprising Connections (you probably didn't know these)
- `geometry → go-geom Polygon/Point Type Override` --semantically_similar_to--> `GeoJSONPolygon Schema`  [INFERRED] [semantically similar]
  sqlc.yml → openapi.yml
- `Dropped ST_Equals Rectangle CHECK Constraint` --conceptually_related_to--> `GeoJSONPolygon Schema`  [INFERRED]
  ARCHITECTURE.md → openapi.yml
- `compose.yml Local Dev PostGIS Service` --conceptually_related_to--> `PostGIS + btree_gist extensions`  [INFERRED]
  compose.yml → AGENT.md
- `Role Derivation via Subtype Table Join` --conceptually_related_to--> `OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)`  [INFERRED]
  ARCHITECTURE.md → openapi.yml
- `No Availability Check — Let Postgres Arbitrate` --conceptually_related_to--> `GET/POST /api/rentals`  [INFERRED]
  ARCHITECTURE.md → openapi.yml

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **CI Parallel Job Pipeline** — github_workflows_ci_build_job, github_workflows_ci_build_docker_job, github_workflows_ci_lint_job, github_workflows_ci_unit_tests_job, github_workflows_ci_integration_tests_job [EXTRACTED 0.90]
- **Notification Fan-out Flow** — agent_notification_service, agent_email_sender, agent_dispatcher, internal_emailtemplates_announcement_html, internal_emailtemplates_broadcast_html, architecture_schwarzes_brett [EXTRACTED 0.90]
- **Handler-Service-Repository Request Pipeline** — claude_skills_three_layer_architecture_handler_layer, claude_skills_three_layer_architecture_service_layer, claude_skills_three_layer_architecture_repository_layer, architecture_main_go_wiring [EXTRACTED 0.90]

## Communities (43 total, 1 thin omitted)

### Community 0 - "net/http.ResponseWriter"
Cohesion: 0.09
Nodes (34): encoding/json.RawMessage, net/http.Request, net/http.ResponseWriter, AuthHandler, createCropRequest, createFieldRequest, createPlotRequest, CropHandler (+26 more)

### Community 1 - "NotificationService"
Cohesion: 0.07
Nodes (40): html/template.Template, io/fs.FS, sync.Map, sync.Once, sync.WaitGroup, testing/fstest.MapFS, broadcastRequest, broadcastResponse (+32 more)

### Community 2 - "AnnouncementWithFarm"
Cohesion: 0.11
Nodes (20): time.Time, AnnouncementHandler, announcementResponse, createAnnouncementRequest, createAnnouncementResponse, NewAnnouncementHandler(), toAnnouncementResponse(), AnnouncementWithFarm (+12 more)

### Community 3 - "AuthService"
Cohesion: 0.09
Nodes (29): time.Duration, HashPassword(), TestHashUsesFreshSalt(), TestHashVerify(), TestVerifyRejectsMalformed(), VerifyPassword(), Claims, Issuer (+21 more)

### Community 4 - "main"
Cohesion: 0.12
Nodes (29): main(), migrateDB(), seedAdmin(), FieldHandler, nearbyPlotResponse, PlotSearchHandler, NewCropHandler(), NewFieldHandler() (+21 more)

### Community 5 - "testing.T"
Cohesion: 0.16
Nodes (45): DBTX, github.com/jackc/pgx/v5/pgxpool.Pool, testing.T, findAccount(), seedOrphanAccount(), TestListAccounts_CarriesNoPasswordMaterial(), TestListAccounts_DerivesRoleFromSubtypeMembership(), TestListAccounts_FiltersNarrowBothItemsAndTotal() (+37 more)

### Community 6 - "Statistics"
Cohesion: 0.10
Nodes (26): accountStatisticsResponse, fieldStatisticsResponse, plotStatisticsResponse, rentalStatisticsResponse, StatisticsHandler, statisticsResponse, NewStatisticsHandler(), toStatisticsResponse() (+18 more)

### Community 7 - "github.com/google/uuid.UUID"
Cohesion: 0.14
Nodes (18): Account, Admin, Announcement, Crop, Customer, Farm, Farmer, Field (+10 more)

### Community 8 - "context.Context"
Cohesion: 0.06
Nodes (24): PostalCode, context.Context, github.com/twpayne/go-geom.Point, Account, AccountListFilter, AccountListing, Role, Crop (+16 more)

### Community 9 - "field"
Cohesion: 0.06
Nodes (32): Rental, RentalWithPlot, RentalWithPlotAndCustomer, mapRentalError(), rentalRepository, account, admin, customer (+24 more)

### Community 10 - "github.com/twpayne/go-geom.Polygon"
Cohesion: 0.17
Nodes (7): github.com/twpayne/go-geom.Polygon, Field, FieldWithPlots, NearbyPlot, Plot, PlotWithCrops, fieldRepository

### Community 11 - "FarmListing"
Cohesion: 0.11
Nodes (24): FarmHandler, farmListingResponse, farmOwnerResponse, farmPageResponse, farmResponse, NewFarmHandler(), toFarmPageResponse(), Farm (+16 more)

### Community 12 - "NewRouter"
Cohesion: 0.13
Nodes (30): net/http.Cookie, net/http.Handler, net/http/httptest.ResponseRecorder, NewRouter(), ClaimsFromContext(), RequireAuth(), assertSameClaims(), assertUnauthorized() (+22 more)

### Community 13 - "Schwarzes Brett (Farmer Announcement Board)"
Cohesion: 0.13
Nodes (19): POST /api/announcements (Schwarzes Brett), services.Dispatcher, EmailSender (internal/repositories/email_sender.go), services.NotificationService, POST /api/notifications, Best-effort Async Delivery After Response, Board-not-Inbox Visibility Design Choice, Embedded Templates (embed.FS) Rationale (+11 more)

### Community 14 - "Three-Layer Architecture Pattern"
Cohesion: 0.12
Nodes (19): config.Load(), testcontainers integration testing, BaaS Backend Architecture, Containerfile Multi-stage Deployment, Known Gaps and Rough Edges (12 items), sqlc + dbmate Migrations-as-schema-of-record, Applying Three-Layer Pattern in Microservices, Three-Layer Architecture Pattern (+11 more)

### Community 15 - "Config"
Cohesion: 0.21
Nodes (16): Config, Load(), parseBoolEnv(), parseIntEnv(), TestParseIntEnv(), TestValidateSMTP_CredentialsMustBeSetInPairs(), TestValidateSMTP_DisabledSkipsTheCredentialRules(), TestValidateSMTP_EnabledRequiresHostAndSender() (+8 more)

### Community 16 - "OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)"
Cohesion: 0.18
Nodes (15): PostGIS KNN Operator <-> with GiST Index, Role Derivation via Subtype Table Join, GET /api/plots/nearest Spatial Search, POST /api/auth/logout, GET /api/auth/me, POST /api/auth/register, Crop Schema, GET/POST /api/crops (+7 more)

### Community 17 - "crop"
Cohesion: 0.14
Nodes (8): GetCropsByPlotsRow, InsertCropParams, InsertPlotCropParams, Queries, crop, field_crop, field_crop, plot_crop

### Community 18 - "email_sender_test.go"
Cohesion: 0.13
Nodes (32): newEmailSender(), bytes.Buffer, mime/multipart.Writer, net/smtp.Auth, addressHeader(), encodeHeader(), needsEncoding(), NewConsoleEmailSender() (+24 more)

### Community 19 - ".ListAccounts"
Cohesion: 0.22
Nodes (14): AccountHandler, accountListingResponse, accountPageResponse, NewAccountHandler(), optionalRole(), toAccountPageResponse(), AccountService, NewAccountService() (+6 more)

### Community 20 - "account.sql.go"
Cohesion: 0.15
Nodes (10): GetAccountByEmailRow, GetAccountByIDRow, GetAllRecipientsRow, InsertAccountParams, InsertAdminParams, InsertCustomerParams, InsertFarmerParams, ListAccountsParams (+2 more)

### Community 21 - "Service Layer (internal/services)"
Cohesion: 0.23
Nodes (12): Dropped ST_Equals Rectangle CHECK Constraint, SQLSTATE-to-HTTP Error Translation Table, mapGeometryError(), trg_plot_within_field Trigger (ST_Within), AccountHandler (example), AccountRepository (example), AccountService (example), Repository-interface Dependency Inversion (+4 more)

### Community 22 - "plot.sql.go"
Cohesion: 0.22
Nodes (7): GetNearestPlotsParams, GetNearestPlotsRow, GetPlotByIDRow, GetPlotsByFieldsRow, InsertPlotParams, InsertPlotRow, Queries

### Community 23 - "GET/POST /api/rentals"
Cohesion: 0.29
Nodes (8): Bauer as a Service (BaaS) Project, chi HTTP router, PostGIS + btree_gist extensions, Backend/Frontend Repo Scope Boundary, No Availability Check — Let Postgres Arbitrate, rental_no_overlap EXCLUDE Constraint (btree_gist), Rental / RentalWithPlot Schema, GET/POST /api/rentals

### Community 24 - "ListFarmsRow"
Cohesion: 0.21
Nodes (7): GetFarmByIDRow, InsertFarmParams, ListFarmsParams, ListFarmsRow, github.com/jackc/pgx/v5/pgtype.Date, github.com/jackc/pgx/v5/pgtype.Int4, Queries

### Community 25 - "GET /api/statistics Role-derived Scope"
Cohesion: 0.40
Nodes (6): DISTINCT Audience Query via rental/plot/field Join, Single-round-trip Aggregate Query (MVCC snapshot consistency), GET /api/statistics Role-derived Scope, GET /api/statistics, Statistics Schema (farm/platform scope), GET /api/statistics (README description)

### Community 26 - "github.com/jackc/pgx/v5/pgtype.Timestamptz"
Cohesion: 0.18
Nodes (9): GetFarmStatisticsRow, GetPlatformStatisticsRow, GetRentalsByCustomerRow, GetRentalsByFarmRow, InsertRentalParams, InsertRentalRow, github.com/jackc/pgx/v5/pgtype.Timestamptz, Queries (+1 more)

### Community 27 - "Login Flow (argon2id + JWT)"
Cohesion: 0.50
Nodes (5): argon2id Password Hashing, Login Flow (argon2id + JWT), HS256 JWT Access/Refresh Tokens, Timing-equaliser dummy hash verification, POST /api/auth/login

### Community 28 - "announcement.sql.go"
Cohesion: 0.25
Nodes (6): GetAnnouncementsByFarmerRow, GetAnnouncementsForCustomerRow, GetCustomersOfFarmerRow, InsertAnnouncementParams, InsertAnnouncementRow, Queries

## Knowledge Gaps
- **25 isolated node(s):** `github.com/Neue-Konzepte-BaaS/backend`, `createAnnouncementRequest`, `loginRequest`, `registerRequest`, `meResponse` (+20 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 69 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `main` to `net/http.ResponseWriter`, `NotificationService`, `AnnouncementWithFarm`, `AuthService`, `testing.T`, `Statistics`, `FarmListing`, `NewRouter`, `Config`, `email_sender_test.go`, `.ListAccounts`?**
  _High betweenness centrality (0.078) - this node is a cross-community bridge._
- **Why does `plot` connect `field` to `crop`?**
  _High betweenness centrality (0.055) - this node is a cross-community bridge._
- **Why does `MustClaimsFromContext()` connect `net/http.ResponseWriter` to `context.Context`, `AuthService`, `NewRouter`?**
  _High betweenness centrality (0.040) - this node is a cross-community bridge._
- **What connects `github.com/Neue-Konzepte-BaaS/backend`, `createAnnouncementRequest`, `loginRequest` to the rest of the system?**
  _25 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.09474206349206349 - nodes in this community are weakly interconnected._
- **Should `NotificationService` be split into smaller, more focused modules?**
  _Cohesion score 0.06868686868686869 - nodes in this community are weakly interconnected._
- **Should `AnnouncementWithFarm` be split into smaller, more focused modules?**
  _Cohesion score 0.11229946524064172 - nodes in this community are weakly interconnected._