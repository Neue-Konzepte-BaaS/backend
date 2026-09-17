# Graph Report - backend  (2026-09-17)

## Corpus Check
- 111 files · ~151,639 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 5 file(s) not represented in the graph (top: (none) 4, .example 1)

## Summary
- 740 nodes · 1782 edges · 44 communities (29 shown, 2 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 106 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `bb954f71`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- net/http.ResponseWriter
- testing.T
- AnnouncementWithFarm
- AuthService
- Crop
- main
- Statistics
- github.com/google/uuid.UUID
- context.Context
- crop
- github.com/twpayne/go-geom.Polygon
- NotificationService
- auth_test.go
- Schwarzes Brett (Farmer Announcement Board)
- Three-Layer Architecture Pattern
- Config
- OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)
- Queries
- email_sender.go
- NewRouter
- account.sql.go
- Service Layer (internal/services)
- Claims
- GET/POST /api/rentals
- InsertFieldParams
- GET /api/statistics Role-derived Scope
- .GetFarmStatistics
- Login Flow (argon2id + JWT)
- .FindPostalCodeCoordinates
- github.com/Neue-Konzepte-BaaS/backend
- .InsertBroadcastNotification

## God Nodes (most connected - your core abstractions)
1. `main()` - 37 edges
2. `Queries` - 22 edges
3. `Account` - 22 edges
4. `WriteError()` - 21 edges
5. `WriteJSON()` - 20 edges
6. `NewRouter()` - 18 edges
7. `MustClaimsFromContext()` - 18 edges
8. `NotificationService` - 16 edges
9. `AnnouncementWithFarm` - 16 edges
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
- **CI Parallel Job Pipeline** — github_workflows_ci_build_job, github_workflows_ci_build_docker_job, github_workflows_ci_lint_job, github_workflows_ci_unit_tests_job, github_workflows_ci_integration_tests_job [EXTRACTED 0.90]
- **Notification Fan-out Flow** — agent_notification_service, agent_email_sender, agent_dispatcher, internal_emailtemplates_announcement_html, internal_emailtemplates_broadcast_html, architecture_schwarzes_brett [EXTRACTED 0.90]
- **Handler-Service-Repository Request Pipeline** — claude_skills_three_layer_architecture_handler_layer, claude_skills_three_layer_architecture_service_layer, claude_skills_three_layer_architecture_repository_layer, architecture_main_go_wiring [EXTRACTED 0.90]

## Communities (44 total, 2 thin omitted)

### Community 0 - "net/http.ResponseWriter"
Cohesion: 0.09
Nodes (32): net/http.Request, net/http.ResponseWriter, AnnouncementHandler, announcementResponse, AuthHandler, createAnnouncementRequest, createAnnouncementResponse, createCropRequest (+24 more)

### Community 1 - "testing.T"
Cohesion: 0.08
Nodes (58): newEmailSender(), Dispatcher, io/fs.FS, sync.WaitGroup, testing/fstest.MapFS, testing.T, HashPassword(), TestHashUsesFreshSalt() (+50 more)

### Community 2 - "AnnouncementWithFarm"
Cohesion: 0.05
Nodes (40): time.Time, InboxHandler, inboxItemResponse, NewInboxHandler(), toInboxItemResponse(), AnnouncementWithFarm, InboxItem, InboxItemKind (+32 more)

### Community 3 - "AuthService"
Cohesion: 0.17
Nodes (18): time.Duration, Issuer, NewIssuer(), TestIssueParseRoundTrip(), TestParseRejectsAlgNone(), TestParseRejectsForeignSecret(), TestParseRejectsWrongType(), AuthService (+10 more)

### Community 4 - "Crop"
Cohesion: 0.14
Nodes (8): Crop, isUniqueViolation(), mapCropError(), toCrops(), CropService, NewCropService(), cropOffered(), cropRepository

### Community 5 - "main"
Cohesion: 0.11
Nodes (36): main(), migrateDB(), seedAdmin(), DBTX, github.com/jackc/pgx/v5/pgxpool.Pool, broadcastRequest, broadcastResponse, NotificationHandler (+28 more)

### Community 6 - "Statistics"
Cohesion: 0.10
Nodes (24): accountStatisticsResponse, fieldStatisticsResponse, plotStatisticsResponse, rentalStatisticsResponse, StatisticsHandler, statisticsResponse, NewStatisticsHandler(), toStatisticsResponse() (+16 more)

### Community 7 - "github.com/google/uuid.UUID"
Cohesion: 0.10
Nodes (28): Account, Admin, Announcement, BroadcastNotification, Crop, Customer, Farm, Farmer (+20 more)

### Community 8 - "context.Context"
Cohesion: 0.17
Nodes (7): context.Context, Account, Role, Recipient, accountRepository, fakeAccountRepo, fakeRecipientRepo

### Community 9 - "crop"
Cohesion: 0.06
Nodes (31): GetNearestPlotsParams, GetNearestPlotsRow, Rental, RentalWithPlot, Queries, mapRentalError(), rentalRepository, account (+23 more)

### Community 10 - "github.com/twpayne/go-geom.Polygon"
Cohesion: 0.07
Nodes (29): encoding/json.RawMessage, github.com/twpayne/go-geom.Polygon, createFieldRequest, createPlotRequest, FieldHandler, fieldResponse, fieldWithPlotsResponse, nearbyPlotResponse (+21 more)

### Community 11 - "NotificationService"
Cohesion: 0.19
Nodes (8): html/template.Template, sync.Map, sync.Once, announcementData, broadcastData, NotificationService, RecipientData, templateEntry

### Community 12 - "auth_test.go"
Cohesion: 0.22
Nodes (18): net/http.Cookie, ClaimsFromContext(), RequireAuth(), assertSameClaims(), assertUnauthorized(), newRequest(), TestClaimsFromContextIgnoresForeignKey(), TestClaimsFromContextWithoutClaims() (+10 more)

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

### Community 17 - "Queries"
Cohesion: 0.22
Nodes (4): GetCropsByPlotsRow, InsertCropParams, InsertPlotCropParams, Queries

### Community 18 - "email_sender.go"
Cohesion: 0.21
Nodes (10): bytes.Buffer, mime/multipart.Writer, net/smtp.Auth, addressHeader(), encodeHeader(), needsEncoding(), writeAttachment(), writeMultipartBody() (+2 more)

### Community 19 - "NewRouter"
Cohesion: 0.14
Nodes (21): AnnouncementHandler, AuthHandler, CropHandler, FarmHandler, FieldHandler, net/http.Handler, net/http/httptest.ResponseRecorder, NewRouter() (+13 more)

### Community 20 - "account.sql.go"
Cohesion: 0.20
Nodes (7): GetAccountByEmailRow, GetAccountByIDRow, GetAllRecipientsRow, InsertAccountParams, InsertCustomerParams, InsertFarmerParams, Queries

### Community 21 - "Service Layer (internal/services)"
Cohesion: 0.23
Nodes (12): Dropped ST_Equals Rectangle CHECK Constraint, SQLSTATE-to-HTTP Error Translation Table, mapGeometryError(), trg_plot_within_field Trigger (ST_Within), AccountHandler (example), AccountRepository (example), AccountService (example), Repository-interface Dependency Inversion (+4 more)

### Community 22 - "Claims"
Cohesion: 0.22
Nodes (5): Claims, RegisterInput, TokenPair, jwt.RegisteredClaims, stubAuthService

### Community 23 - "GET/POST /api/rentals"
Cohesion: 0.29
Nodes (8): Bauer as a Service (BaaS) Project, chi HTTP router, PostGIS + btree_gist extensions, Backend/Frontend Repo Scope Boundary, No Availability Check — Let Postgres Arbitrate, rental_no_overlap EXCLUDE Constraint (btree_gist), Rental / RentalWithPlot Schema, GET/POST /api/rentals

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

## Knowledge Gaps
- **31 isolated node(s):** `broadcastData`, `announcementData`, `FarmRepository`, `FieldRepository`, `PlotRepository` (+26 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 81 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `main` to `net/http.ResponseWriter`, `testing.T`, `AnnouncementWithFarm`, `AuthService`, `Crop`, `Statistics`, `github.com/twpayne/go-geom.Polygon`, `Config`, `NewRouter`?**
  _High betweenness centrality (0.109) - this node is a cross-community bridge._
- **Why does `MustClaimsFromContext()` connect `net/http.ResponseWriter` to `context.Context`, `NewRouter`, `auth_test.go`, `Claims`?**
  _High betweenness centrality (0.042) - this node is a cross-community bridge._
- **Why does `seedAdmin()` connect `main` to `context.Context`, `testing.T`, `AuthService`, `Config`?**
  _High betweenness centrality (0.033) - this node is a cross-community bridge._
- **What connects `broadcastData`, `announcementData`, `FarmRepository` to the rest of the system?**
  _31 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.09351256575102279 - nodes in this community are weakly interconnected._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.0814111261872456 - nodes in this community are weakly interconnected._
- **Should `AnnouncementWithFarm` be split into smaller, more focused modules?**
  _Cohesion score 0.05201266395296246 - nodes in this community are weakly interconnected._