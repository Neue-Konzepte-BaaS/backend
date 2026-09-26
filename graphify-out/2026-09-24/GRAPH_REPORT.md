# Graph Report - backend  (2026-09-17)

## Corpus Check
- 120 files · ~163,868 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 5 file(s) not represented in the graph (top: (none) 4, .example 1)

## Summary
- 953 nodes · 2535 edges · 53 communities (35 shown, 3 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 163 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `3b52c553`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- net/http.ResponseWriter
- NotificationService
- AnnouncementWithFarm
- AuthService
- RentalService
- testing.T
- Statistics
- github.com/google/uuid.UUID
- Account
- field
- github.com/twpayne/go-geom.Polygon
- FarmListing
- NewRouter
- Schwarzes Brett (Farmer Announcement Board)
- Three-Layer Architecture Pattern
- Config
- OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)
- Queries
- email_sender_test.go
- RentalHandler
- account.sql.go
- Service Layer (internal/services)
- plot.sql.go
- GET/POST /api/rentals
- ListFarmsRow
- GET /api/statistics Role-derived Scope
- .GetFarmStatistics
- Login Flow (argon2id + JWT)
- announcement.sql.go
- github.com/Neue-Konzepte-BaaS/backend
- .InsertBroadcastNotification
- context.Context
- Rental
- BroadcastNotification
- GetFieldByIDRow
- PlotSearchService
- .FindPostalCodeCoordinates
- rental

## God Nodes (most connected - your core abstractions)
1. `main()` - 42 edges
2. `Account` - 29 edges
3. `New()` - 28 edges
4. `WriteError()` - 27 edges
5. `WriteJSON()` - 25 edges
6. `Queries` - 24 edges
7. `MustClaimsFromContext()` - 22 edges
8. `setupTestDB()` - 21 edges
9. `Role` - 21 edges
10. `Crop` - 20 edges

## Surprising Connections (you probably didn't know these)
- `geometry → go-geom Polygon/Point Type Override` --semantically_similar_to--> `GeoJSONPolygon Schema`  [INFERRED] [semantically similar]
  sqlc.yml → openapi.yml
- `GeoJSONPolygon Schema` --conceptually_related_to--> `Dropped ST_Equals Rectangle CHECK Constraint`  [INFERRED]
  openapi.yml → ARCHITECTURE.md
- `compose.yml Local Dev PostGIS Service` --conceptually_related_to--> `PostGIS + btree_gist extensions`  [INFERRED]
  compose.yml → AGENT.md
- `OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)` --conceptually_related_to--> `Role Derivation via Subtype Table Join`  [INFERRED]
  openapi.yml → ARCHITECTURE.md
- `GET/POST /api/rentals` --conceptually_related_to--> `No Availability Check — Let Postgres Arbitrate`  [INFERRED]
  openapi.yml → ARCHITECTURE.md

## Import Cycles
- None detected.

## Communities (53 total, 3 thin omitted)

### Community 0 - "net/http.ResponseWriter"
Cohesion: 0.09
Nodes (36): cropResponse, encoding/json.RawMessage, net/http.Request, net/http.ResponseWriter, AuthHandler, createCropRequest, createFieldRequest, createPlotRequest (+28 more)

### Community 1 - "NotificationService"
Cohesion: 0.07
Nodes (45): Dispatcher, html/template.Template, io/fs.FS, sync.Map, sync.Once, sync.WaitGroup, testing/fstest.MapFS, broadcastRequest (+37 more)

### Community 2 - "AnnouncementWithFarm"
Cohesion: 0.08
Nodes (33): time.Time, AnnouncementHandler, announcementResponse, createAnnouncementRequest, createAnnouncementResponse, InboxHandler, inboxItemResponse, NewAnnouncementHandler() (+25 more)

### Community 3 - "AuthService"
Cohesion: 0.10
Nodes (25): time.Duration, HashPassword(), TestHashUsesFreshSalt(), TestHashVerify(), TestVerifyRejectsMalformed(), VerifyPassword(), Issuer, NewIssuer() (+17 more)

### Community 4 - "RentalService"
Cohesion: 0.12
Nodes (24): fakeFarmRepo, NewFieldHandler(), CropService, NewCropService(), FieldService, PlotService, NewFieldService(), NewPlotService() (+16 more)

### Community 5 - "testing.T"
Cohesion: 0.11
Nodes (54): main(), migrateDB(), seedAdmin(), DBTX, github.com/jackc/pgx/v5/pgxpool.Pool, testing.T, findAccount(), seedOrphanAccount() (+46 more)

### Community 6 - "Statistics"
Cohesion: 0.12
Nodes (26): accountStatisticsResponse, fieldStatisticsResponse, plotStatisticsResponse, rentalStatisticsResponse, StatisticsHandler, statisticsResponse, NewStatisticsHandler(), toStatisticsResponse() (+18 more)

### Community 7 - "github.com/google/uuid.UUID"
Cohesion: 0.10
Nodes (26): Account, Admin, Announcement, BroadcastNotification, Customer, Farm, Farmer, Field (+18 more)

### Community 8 - "Account"
Cohesion: 0.07
Nodes (28): AccountHandler, accountListingResponse, accountPageResponse, NewAccountHandler(), optionalRole(), toAccountPageResponse(), Account, AccountListFilter (+20 more)

### Community 9 - "field"
Cohesion: 0.07
Nodes (31): account, admin, customer, farmer, idx_account_email, check_plot_within_field(), field, plot (+23 more)

### Community 10 - "github.com/twpayne/go-geom.Polygon"
Cohesion: 0.10
Nodes (10): github.com/twpayne/go-geom.Polygon, Field, FieldWithPlots, NearbyPlot, Plot, mapGeometryError(), PlotWithCrops, fieldRepository (+2 more)

### Community 11 - "FarmListing"
Cohesion: 0.10
Nodes (28): FieldStatistics, fieldStatisticsResponse, FarmHandler, farmListingResponse, farmOwnerResponse, farmPageResponse, farmResponse, NewFarmHandler() (+20 more)

### Community 12 - "NewRouter"
Cohesion: 0.07
Nodes (43): AccountHandler, AnnouncementHandler, AuthHandler, CropHandler, FarmHandler, FieldHandler, net/http.Cookie, net/http.Handler (+35 more)

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
Cohesion: 0.24
Nodes (6): GetFarmByIDRow, InsertFarmParams, ListFarmsParams, ListFarmsRow, github.com/jackc/pgx/v5/pgtype.Int4, Queries

### Community 25 - "GET /api/statistics Role-derived Scope"
Cohesion: 0.40
Nodes (6): DISTINCT Audience Query via rental/plot/field Join, Single-round-trip Aggregate Query (MVCC snapshot consistency), GET /api/statistics Role-derived Scope, GET /api/statistics, Statistics Schema (farm/platform scope), GET /api/statistics (README description)

### Community 26 - ".GetFarmStatistics"
Cohesion: 0.40
Nodes (3): GetFarmStatisticsRow, GetPlatformStatisticsRow, Queries

### Community 27 - "Login Flow (argon2id + JWT)"
Cohesion: 0.50
Nodes (5): argon2id Password Hashing, Login Flow (argon2id + JWT), HS256 JWT Access/Refresh Tokens, Timing-equaliser dummy hash verification, POST /api/auth/login

### Community 28 - "announcement.sql.go"
Cohesion: 0.25
Nodes (6): GetAnnouncementsByFarmerRow, GetAnnouncementsForCustomerRow, GetCustomersOfFarmerRow, InsertAnnouncementParams, InsertAnnouncementRow, Queries

### Community 42 - ".InsertBroadcastNotification"
Cohesion: 0.32
Nodes (4): InsertBroadcastNotificationParams, Queries, broadcast_notification, idx_broadcast_notification_created

### Community 44 - "context.Context"
Cohesion: 0.08
Nodes (6): context.Context, Crop, mapCropError(), toCrops(), cropRepository, fakeCropRepo

### Community 45 - "Rental"
Cohesion: 0.15
Nodes (10): Plot, Rental, RentalStatus, RentalWithField, RentalWithPlot, RentalWithPlotAndCustomer, mapRentalError(), Recipient (+2 more)

### Community 46 - "BroadcastNotification"
Cohesion: 0.27
Nodes (5): BroadcastNotification, toModelBroadcastNotification(), broadcastNotificationRepository, fakeBroadcastRepo, fakeInboxBroadcastRepo

### Community 47 - "GetFieldByIDRow"
Cohesion: 0.28
Nodes (4): GetFieldByIDRow, GetFieldsByFarmRow, InsertFieldParams, Queries

### Community 48 - "PlotSearchService"
Cohesion: 0.53
Nodes (4): PlotSearchHandler, NewPlotSearchHandler(), PlotSearchService, NewPlotSearchService()

### Community 49 - ".FindPostalCodeCoordinates"
Cohesion: 0.50
Nodes (3): PostalCode, github.com/twpayne/go-geom.Point, Queries

## Knowledge Gaps
- **27 isolated node(s):** `rentPlotRequest`, `BroadcastNotificationRepository`, `PostalCodeRepository`, `createCropRequest`, `setPlotCropsRequest` (+22 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 92 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `testing.T` to `net/http.ResponseWriter`, `NotificationService`, `AnnouncementWithFarm`, `AuthService`, `RentalService`, `Statistics`, `Account`, `FarmListing`, `NewRouter`, `Config`, `PlotSearchService`, `email_sender_test.go`?**
  _High betweenness centrality (0.085) - this node is a cross-community bridge._
- **Why does `crop` connect `field` to `Queries`, `github.com/twpayne/go-geom.Polygon`?**
  _High betweenness centrality (0.072) - this node is a cross-community bridge._
- **What connects `rentPlotRequest`, `BroadcastNotificationRepository`, `PostalCodeRepository` to the rest of the system?**
  _27 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.09207161125319693 - nodes in this community are weakly interconnected._
- **Should `NotificationService` be split into smaller, more focused modules?**
  _Cohesion score 0.07062146892655367 - nodes in this community are weakly interconnected._
- **Should `AnnouncementWithFarm` be split into smaller, more focused modules?**
  _Cohesion score 0.07770582793709528 - nodes in this community are weakly interconnected._
- **Should `AuthService` be split into smaller, more focused modules?**
  _Cohesion score 0.1036036036036036 - nodes in this community are weakly interconnected._