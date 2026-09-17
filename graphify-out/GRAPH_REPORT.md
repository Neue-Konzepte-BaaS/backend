# Graph Report - backend  (2026-09-17)

## Corpus Check
- 118 files · ~161,166 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 5 file(s) not represented in the graph (top: (none) 4, .example 1)

## Summary
- 876 nodes · 2229 edges · 52 communities (35 shown, 3 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 153 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `33ab8c60`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- net/http.ResponseWriter
- notification_service_test.go
- BroadcastNotification
- AuthService
- Crop
- testing.T
- Statistics
- github.com/google/uuid.UUID
- context.Context
- field
- github.com/twpayne/go-geom.Polygon
- Account
- FarmListing
- Schwarzes Brett (Farmer Announcement Board)
- Three-Layer Architecture Pattern
- Config
- OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)
- Queries
- email_sender_test.go
- NewRouter
- account.sql.go
- Service Layer (internal/services)
- time.Time
- GET/POST /api/rentals
- GetFieldByIDRow
- GET /api/statistics Role-derived Scope
- .GetFarmStatistics
- Login Flow (argon2id + JWT)
- .FindPostalCodeCoordinates
- github.com/Neue-Konzepte-BaaS/backend
- .InsertBroadcastNotification
- AnnouncementWithFarm
- plot.sql.go
- ListFarmsRow
- Rental
- notification_handler.go
- FarmHandler
- PlotSearchHandler

## God Nodes (most connected - your core abstractions)
1. `main()` - 42 edges
2. `Account` - 29 edges
3. `New()` - 26 edges
4. `Queries` - 24 edges
5. `WriteError()` - 24 edges
6. `WriteJSON()` - 23 edges
7. `Role` - 21 edges
8. `MustClaimsFromContext()` - 20 edges
9. `setupTestDB()` - 20 edges
10. `NewRouter()` - 19 edges

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

## Communities (52 total, 3 thin omitted)

### Community 0 - "net/http.ResponseWriter"
Cohesion: 0.09
Nodes (39): encoding/json.RawMessage, net/http.Request, net/http.ResponseWriter, AuthHandler, createCropRequest, createFieldRequest, createPlotRequest, CropHandler (+31 more)

### Community 1 - "notification_service_test.go"
Cohesion: 0.07
Nodes (39): Dispatcher, html/template.Template, io/fs.FS, sync.Map, sync.Once, sync.WaitGroup, testing/fstest.MapFS, NewDispatcher() (+31 more)

### Community 2 - "BroadcastNotification"
Cohesion: 0.09
Nodes (21): InboxHandler, NewInboxHandler(), BroadcastNotification, toModelBroadcastNotification(), InboxService, NewInboxService(), TestGetInboxForCustomer_AnnouncementFailureIsReturned(), TestGetInboxForCustomer_BroadcastFailureIsReturned() (+13 more)

### Community 3 - "AuthService"
Cohesion: 0.09
Nodes (29): time.Duration, HashPassword(), TestHashUsesFreshSalt(), TestHashVerify(), TestVerifyRejectsMalformed(), VerifyPassword(), Claims, Issuer (+21 more)

### Community 4 - "Crop"
Cohesion: 0.19
Nodes (6): Crop, mapCropError(), toCrops(), CropService, NewCropService(), cropRepository

### Community 5 - "testing.T"
Cohesion: 0.11
Nodes (52): main(), migrateDB(), seedAdmin(), DBTX, github.com/jackc/pgx/v5/pgxpool.Pool, testing.T, findAccount(), seedOrphanAccount() (+44 more)

### Community 6 - "Statistics"
Cohesion: 0.10
Nodes (25): accountStatisticsResponse, fieldStatisticsResponse, plotStatisticsResponse, rentalStatisticsResponse, StatisticsHandler, statisticsResponse, NewStatisticsHandler(), toStatisticsResponse() (+17 more)

### Community 7 - "github.com/google/uuid.UUID"
Cohesion: 0.11
Nodes (26): Account, Admin, Announcement, BroadcastNotification, Customer, Farm, Farmer, Field (+18 more)

### Community 8 - "context.Context"
Cohesion: 0.11
Nodes (4): context.Context, Recipient, fakeRecipientRepo, fakeStatisticsRepo

### Community 9 - "field"
Cohesion: 0.07
Nodes (31): cropResponse, nearbyPlotResponse, PlotSearchHandler, NewPlotSearchHandler(), NearbyPlot, PlotSearchService, NewPlotSearchService(), account (+23 more)

### Community 10 - "github.com/twpayne/go-geom.Polygon"
Cohesion: 0.14
Nodes (8): github.com/twpayne/go-geom.Polygon, Field, FieldWithPlots, Plot, mapGeometryError(), PlotWithCrops, fieldRepository, plotRepository

### Community 11 - "Account"
Cohesion: 0.08
Nodes (26): AccountHandler, accountListingResponse, accountPageResponse, NewAccountHandler(), optionalRole(), toAccountPageResponse(), Account, AccountListFilter (+18 more)

### Community 12 - "FarmListing"
Cohesion: 0.10
Nodes (28): FieldStatistics, fieldStatisticsResponse, FarmHandler, farmListingResponse, farmOwnerResponse, farmPageResponse, farmResponse, NewFarmHandler() (+20 more)

### Community 13 - "Schwarzes Brett (Farmer Announcement Board)"
Cohesion: 0.13
Nodes (19): POST /api/announcements (Schwarzes Brett), services.Dispatcher, EmailSender (internal/repositories/email_sender.go), services.NotificationService, POST /api/notifications, Best-effort Async Delivery After Response, Board-not-Inbox Visibility Design Choice, Embedded Templates (embed.FS) Rationale (+11 more)

### Community 14 - "Three-Layer Architecture Pattern"
Cohesion: 0.12
Nodes (19): config.Load(), testcontainers integration testing, BaaS Backend Architecture, Containerfile Multi-stage Deployment, Known Gaps and Rough Edges (12 items), sqlc + dbmate Migrations-as-schema-of-record, Applying Three-Layer Pattern in Microservices, Three-Layer Architecture Pattern (+11 more)

### Community 15 - "Config"
Cohesion: 0.15
Nodes (20): loginRequest, meResponse, registerRequest, Config, Load(), parseBoolEnv(), parseIntEnv(), TestParseIntEnv() (+12 more)

### Community 16 - "OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)"
Cohesion: 0.18
Nodes (15): PostGIS KNN Operator <-> with GiST Index, Role Derivation via Subtype Table Join, GET /api/plots/nearest Spatial Search, POST /api/auth/logout, GET /api/auth/me, POST /api/auth/register, Crop Schema, GET/POST /api/crops (+7 more)

### Community 17 - "Queries"
Cohesion: 0.19
Nodes (5): Crop, GetCropsByPlotsRow, InsertCropParams, InsertPlotCropParams, Queries

### Community 18 - "email_sender_test.go"
Cohesion: 0.13
Nodes (32): newEmailSender(), bytes.Buffer, mime/multipart.Writer, net/smtp.Auth, addressHeader(), encodeHeader(), needsEncoding(), NewConsoleEmailSender() (+24 more)

### Community 19 - "NewRouter"
Cohesion: 0.09
Nodes (38): AnnouncementHandler, AuthHandler, CropHandler, FieldHandler, net/http.Cookie, net/http.Handler, net/http/httptest.ResponseRecorder, InboxHandler (+30 more)

### Community 20 - "account.sql.go"
Cohesion: 0.15
Nodes (10): GetAccountByEmailRow, GetAccountByIDRow, GetAllRecipientsRow, InsertAccountParams, InsertAdminParams, InsertCustomerParams, InsertFarmerParams, ListAccountsParams (+2 more)

### Community 21 - "Service Layer (internal/services)"
Cohesion: 0.23
Nodes (12): Dropped ST_Equals Rectangle CHECK Constraint, SQLSTATE-to-HTTP Error Translation Table, mapGeometryError(), trg_plot_within_field Trigger (ST_Within), AccountHandler (example), AccountRepository (example), AccountService (example), Repository-interface Dependency Inversion (+4 more)

### Community 22 - "time.Time"
Cohesion: 0.14
Nodes (18): time.Time, AnnouncementHandler, announcementResponse, createAnnouncementRequest, createAnnouncementResponse, inboxItemResponse, NewAnnouncementHandler(), toInboxItemResponse() (+10 more)

### Community 23 - "GET/POST /api/rentals"
Cohesion: 0.29
Nodes (8): Bauer as a Service (BaaS) Project, chi HTTP router, PostGIS + btree_gist extensions, Backend/Frontend Repo Scope Boundary, No Availability Check — Let Postgres Arbitrate, rental_no_overlap EXCLUDE Constraint (btree_gist), Rental / RentalWithPlot Schema, GET/POST /api/rentals

### Community 24 - "GetFieldByIDRow"
Cohesion: 0.28
Nodes (4): GetFieldByIDRow, GetFieldsByFarmRow, InsertFieldParams, Queries

### Community 25 - "GET /api/statistics Role-derived Scope"
Cohesion: 0.40
Nodes (6): DISTINCT Audience Query via rental/plot/field Join, Single-round-trip Aggregate Query (MVCC snapshot consistency), GET /api/statistics Role-derived Scope, GET /api/statistics, Statistics Schema (farm/platform scope), GET /api/statistics (README description)

### Community 26 - ".GetFarmStatistics"
Cohesion: 0.40
Nodes (3): GetFarmStatisticsRow, GetPlatformStatisticsRow, Queries

### Community 27 - "Login Flow (argon2id + JWT)"
Cohesion: 0.50
Nodes (5): argon2id Password Hashing, Login Flow (argon2id + JWT), HS256 JWT Access/Refresh Tokens, Timing-equaliser dummy hash verification, POST /api/auth/login

### Community 28 - ".FindPostalCodeCoordinates"
Cohesion: 0.50
Nodes (3): PostalCode, github.com/twpayne/go-geom.Point, Queries

### Community 42 - ".InsertBroadcastNotification"
Cohesion: 0.32
Nodes (4): InsertBroadcastNotificationParams, Queries, broadcast_notification, idx_broadcast_notification_created

### Community 44 - "AnnouncementWithFarm"
Cohesion: 0.22
Nodes (4): AnnouncementWithFarm, toModelAnnouncement(), Announcement, announcementRepository

### Community 45 - "plot.sql.go"
Cohesion: 0.22
Nodes (7): GetNearestPlotsParams, GetNearestPlotsRow, GetPlotByIDRow, GetPlotsByFieldsRow, InsertPlotParams, InsertPlotRow, Queries

### Community 46 - "ListFarmsRow"
Cohesion: 0.21
Nodes (7): GetFarmByIDRow, InsertFarmParams, ListFarmsParams, ListFarmsRow, github.com/jackc/pgx/v5/pgtype.Date, github.com/jackc/pgx/v5/pgtype.Int4, Queries

### Community 47 - "Rental"
Cohesion: 0.24
Nodes (5): Rental, RentalWithPlot, mapRentalError(), cropOffered(), rentalRepository

### Community 48 - "notification_handler.go"
Cohesion: 0.50
Nodes (4): broadcastRequest, broadcastResponse, NotificationHandler, NewNotificationHandler()

## Knowledge Gaps
- **31 isolated node(s):** `farmResponse`, `BroadcastNotificationRepository`, `FieldRepository`, `PlotRepository`, `RentalRepository` (+26 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 91 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `testing.T` to `net/http.ResponseWriter`, `notification_service_test.go`, `BroadcastNotification`, `AuthService`, `Crop`, `Statistics`, `field`, `Account`, `FarmListing`, `Config`, `notification_handler.go`, `email_sender_test.go`, `NewRouter`, `time.Time`?**
  _High betweenness centrality (0.104) - this node is a cross-community bridge._
- **Why does `RentalWithPlot` connect `Rental` to `field`?**
  _High betweenness centrality (0.056) - this node is a cross-community bridge._
- **Why does `plot` connect `field` to `Rental`?**
  _High betweenness centrality (0.049) - this node is a cross-community bridge._
- **What connects `farmResponse`, `BroadcastNotificationRepository`, `FieldRepository` to the rest of the system?**
  _31 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.08525149190110827 - nodes in this community are weakly interconnected._
- **Should `notification_service_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07256894049346879 - nodes in this community are weakly interconnected._
- **Should `BroadcastNotification` be split into smaller, more focused modules?**
  _Cohesion score 0.09274193548387097 - nodes in this community are weakly interconnected._