# Graph Report - backend  (2026-09-14)

## Corpus Check
- 98 files · ~145,210 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 4 file(s) not represented in the graph (top: (none) 3, .example 1)

## Summary
- 679 nodes · 1693 edges · 42 communities (28 shown, 2 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 99 edges (avg confidence: 0.85)
- Token cost: 112,573 input · 0 output

## Community Hubs (Navigation)
- Announcement & Auth Handlers
- Email Sender Tests
- Announcement Domain
- Password & Token Credentials
- Field/Crop Handlers & Services
- App Bootstrap & DB Wiring
- Statistics Handlers
- DB Query Rows (Core Entities)
- Account & Notification Models
- Accounts/Fields SQL Migrations
- Field/Plot Geometry Models
- Notification Broadcast Handling
- Auth Middleware
- Notification Architecture Rationale
- Deployment & Config Architecture
- App Config Loading
- OpenAPI Field/Crop Endpoints
- Crop SQL Queries
- Email Sender Implementation
- Routing & Role Middleware
- Account SQL Queries
- Three-Layer Architecture Doc
- Plot SQL Queries
- Rental Concurrency & Schema
- Field SQL Queries
- Statistics Architecture Rationale
- Statistics SQL Queries
- Login Flow Architecture
- Postal Code Geo Queries
- Go Module Root

## God Nodes (most connected - your core abstractions)
1. `main()` - 33 edges
2. `Account` - 21 edges
3. `Queries` - 20 edges
4. `NotificationService` - 20 edges
5. `WriteError()` - 20 edges
6. `WriteJSON()` - 19 edges
7. `MustClaimsFromContext()` - 17 edges
8. `NewRouter()` - 16 edges
9. `Statistics` - 15 edges
10. `NewSMTPEmailSender()` - 15 edges

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
- **Handler-Service-Repository Request Pipeline** — claude_skills_three_layer_architecture_handler_layer, claude_skills_three_layer_architecture_service_layer, claude_skills_three_layer_architecture_repository_layer, architecture_main_go_wiring [EXTRACTED 0.90]
- **Notification Fan-out Flow** — agent_notification_service, agent_email_sender, agent_dispatcher, internal_emailtemplates_announcement_html, internal_emailtemplates_broadcast_html, architecture_schwarzes_brett [EXTRACTED 0.90]
- **CI Parallel Job Pipeline** — github_workflows_ci_build_job, github_workflows_ci_build_docker_job, github_workflows_ci_lint_job, github_workflows_ci_unit_tests_job, github_workflows_ci_integration_tests_job [EXTRACTED 0.90]

## Communities (42 total, 2 thin omitted)

### Community 0 - "Announcement & Auth Handlers"
Cohesion: 0.09
Nodes (36): encoding/json.RawMessage, net/http.Request, net/http.ResponseWriter, AuthHandler, createCropRequest, createFieldRequest, createPlotRequest, CropHandler (+28 more)

### Community 1 - "Email Sender Tests"
Cohesion: 0.10
Nodes (49): newEmailSender(), sync.WaitGroup, testing/fstest.MapFS, testing.T, NewConsoleEmailSender(), NewSMTPEmailSender(), headerBlock(), headerLine() (+41 more)

### Community 2 - "Announcement Domain"
Cohesion: 0.08
Nodes (24): time.Time, AnnouncementHandler, announcementResponse, createAnnouncementRequest, createAnnouncementResponse, NewAnnouncementHandler(), toAnnouncementResponse(), AnnouncementWithFarm (+16 more)

### Community 3 - "Password & Token Credentials"
Cohesion: 0.10
Nodes (27): HashPassword(), TestHashUsesFreshSalt(), TestHashVerify(), TestVerifyRejectsMalformed(), VerifyPassword(), Claims, Issuer, NewIssuer() (+19 more)

### Community 4 - "Field/Crop Handlers & Services"
Cohesion: 0.10
Nodes (23): FieldHandler, NewFieldHandler(), Crop, isUniqueViolation(), mapCropError(), toCrops(), CropService, NewCropService() (+15 more)

### Community 5 - "App Bootstrap & DB Wiring"
Cohesion: 0.15
Nodes (29): main(), migrateDB(), DBTX, github.com/jackc/pgx/v5/pgxpool.Pool, NewAccountRepository(), rentPast(), seedFarmerWithPlots(), TestGetAnnouncementsForCustomer_OnlyFromFarmersCurrentlyRentedFrom() (+21 more)

### Community 6 - "Statistics Handlers"
Cohesion: 0.10
Nodes (25): accountStatisticsResponse, fieldStatisticsResponse, plotStatisticsResponse, rentalStatisticsResponse, StatisticsHandler, statisticsResponse, NewStatisticsHandler(), toStatisticsResponse() (+17 more)

### Community 7 - "DB Query Rows (Core Entities)"
Cohesion: 0.12
Nodes (24): Account, Admin, Announcement, Crop, Customer, Farmer, Field, GetAnnouncementsByFarmerRow (+16 more)

### Community 8 - "Account & Notification Models"
Cohesion: 0.15
Nodes (8): context.Context, time.Duration, Account, Role, Recipient, accountRepository, fakeAccountRepo, fakeRecipientRepo

### Community 9 - "Accounts/Fields SQL Migrations"
Cohesion: 0.10
Nodes (22): account, admin, customer, farmer, idx_account_email, check_plot_within_field(), field, plot (+14 more)

### Community 10 - "Field/Plot Geometry Models"
Cohesion: 0.12
Nodes (9): github.com/twpayne/go-geom.Polygon, Field, FieldWithPlots, NearbyPlot, Plot, mapGeometryError(), PlotWithCrops, fieldRepository (+1 more)

### Community 11 - "Notification Broadcast Handling"
Cohesion: 0.13
Nodes (13): html/template.Template, io/fs.FS, sync.Map, sync.Once, broadcastRequest, broadcastResponse, NotificationHandler, NewNotificationHandler() (+5 more)

### Community 12 - "Auth Middleware"
Cohesion: 0.22
Nodes (18): net/http.Cookie, ClaimsFromContext(), RequireAuth(), assertSameClaims(), assertUnauthorized(), newRequest(), TestClaimsFromContextIgnoresForeignKey(), TestClaimsFromContextWithoutClaims() (+10 more)

### Community 13 - "Notification Architecture Rationale"
Cohesion: 0.13
Nodes (19): POST /api/announcements (Schwarzes Brett), services.Dispatcher, EmailSender (internal/repositories/email_sender.go), services.NotificationService, POST /api/notifications, Best-effort Async Delivery After Response, Board-not-Inbox Visibility Design Choice, Embedded Templates (embed.FS) Rationale (+11 more)

### Community 14 - "Deployment & Config Architecture"
Cohesion: 0.12
Nodes (19): config.Load(), testcontainers integration testing, BaaS Backend Architecture, Containerfile Multi-stage Deployment, Known Gaps and Rough Edges (12 items), sqlc + dbmate Migrations-as-schema-of-record, Applying Three-Layer Pattern in Microservices, Three-Layer Architecture Pattern (+11 more)

### Community 15 - "App Config Loading"
Cohesion: 0.21
Nodes (16): Config, Load(), parseBoolEnv(), parseIntEnv(), TestParseIntEnv(), TestValidateSMTP_CredentialsMustBeSetInPairs(), TestValidateSMTP_DisabledSkipsTheCredentialRules(), TestValidateSMTP_EnabledRequiresHostAndSender() (+8 more)

### Community 16 - "OpenAPI Field/Crop Endpoints"
Cohesion: 0.18
Nodes (15): PostGIS KNN Operator <-> with GiST Index, Role Derivation via Subtype Table Join, GET /api/plots/nearest Spatial Search, POST /api/auth/logout, GET /api/auth/me, POST /api/auth/register, Crop Schema, GET/POST /api/crops (+7 more)

### Community 17 - "Crop SQL Queries"
Cohesion: 0.18
Nodes (6): GetCropsByPlotsRow, InsertCropParams, InsertPlotCropParams, Queries, crop, field_crop

### Community 18 - "Email Sender Implementation"
Cohesion: 0.20
Nodes (11): bytes.Buffer, mime/multipart.Writer, net/smtp.Auth, addressHeader(), encodeHeader(), needsEncoding(), writeAttachment(), writeMultipartBody() (+3 more)

### Community 19 - "Routing & Role Middleware"
Cohesion: 0.29
Nodes (12): net/http.Handler, net/http/httptest.ResponseRecorder, NewRouter(), RequireAnyRole(), RequireRole(), assertForbidden(), requestWithRole(), TestRequireAnyRole_AdmitsEachListedRole() (+4 more)

### Community 20 - "Account SQL Queries"
Cohesion: 0.20
Nodes (7): GetAccountByEmailRow, GetAccountByIDRow, GetAllRecipientsRow, InsertAccountParams, InsertCustomerParams, InsertFarmerParams, Queries

### Community 21 - "Three-Layer Architecture Doc"
Cohesion: 0.23
Nodes (12): Dropped ST_Equals Rectangle CHECK Constraint, SQLSTATE-to-HTTP Error Translation Table, mapGeometryError(), trg_plot_within_field Trigger (ST_Within), AccountHandler (example), AccountRepository (example), AccountService (example), Repository-interface Dependency Inversion (+4 more)

### Community 22 - "Plot SQL Queries"
Cohesion: 0.28
Nodes (4): GetNearestPlotsParams, GetNearestPlotsRow, InsertPlotParams, Queries

### Community 23 - "Rental Concurrency & Schema"
Cohesion: 0.29
Nodes (8): Bauer as a Service (BaaS) Project, chi HTTP router, PostGIS + btree_gist extensions, Backend/Frontend Repo Scope Boundary, No Availability Check — Let Postgres Arbitrate, rental_no_overlap EXCLUDE Constraint (btree_gist), Rental / RentalWithPlot Schema, GET/POST /api/rentals

### Community 25 - "Statistics Architecture Rationale"
Cohesion: 0.40
Nodes (6): DISTINCT Audience Query via rental/plot/field Join, Single-round-trip Aggregate Query (MVCC snapshot consistency), GET /api/statistics Role-derived Scope, GET /api/statistics, Statistics Schema (farm/platform scope), GET /api/statistics (README description)

### Community 26 - "Statistics SQL Queries"
Cohesion: 0.40
Nodes (3): GetFarmStatisticsRow, GetPlatformStatisticsRow, Queries

### Community 27 - "Login Flow Architecture"
Cohesion: 0.50
Nodes (5): argon2id Password Hashing, Login Flow (argon2id + JWT), HS256 JWT Access/Refresh Tokens, Timing-equaliser dummy hash verification, POST /api/auth/login

### Community 28 - "Postal Code Geo Queries"
Cohesion: 0.50
Nodes (3): PostalCode, github.com/twpayne/go-geom.Point, Queries

## Knowledge Gaps
- **24 isolated node(s):** `github.com/Neue-Konzepte-BaaS/backend`, `createAnnouncementRequest`, `loginRequest`, `registerRequest`, `meResponse` (+19 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 62 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `App Bootstrap & DB Wiring` to `Announcement & Auth Handlers`, `Email Sender Tests`, `Announcement Domain`, `Password & Token Credentials`, `Field/Crop Handlers & Services`, `Statistics Handlers`, `Notification Broadcast Handling`, `App Config Loading`, `Routing & Role Middleware`?**
  _High betweenness centrality (0.083) - this node is a cross-community bridge._
- **Why does `MustClaimsFromContext()` connect `Announcement & Auth Handlers` to `Account & Notification Models`, `Routing & Role Middleware`, `Password & Token Credentials`, `Auth Middleware`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **Why does `NotificationService` connect `Notification Broadcast Handling` to `Email Sender Tests`, `Announcement Domain`, `Password & Token Credentials`, `DB Query Rows (Core Entities)`?**
  _High betweenness centrality (0.036) - this node is a cross-community bridge._
- **What connects `github.com/Neue-Konzepte-BaaS/backend`, `createAnnouncementRequest`, `loginRequest` to the rest of the system?**
  _24 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Announcement & Auth Handlers` be split into smaller, more focused modules?**
  _Cohesion score 0.0907103825136612 - nodes in this community are weakly interconnected._
- **Should `Email Sender Tests` be split into smaller, more focused modules?**
  _Cohesion score 0.10324675324675325 - nodes in this community are weakly interconnected._
- **Should `Announcement Domain` be split into smaller, more focused modules?**
  _Cohesion score 0.08305647840531562 - nodes in this community are weakly interconnected._