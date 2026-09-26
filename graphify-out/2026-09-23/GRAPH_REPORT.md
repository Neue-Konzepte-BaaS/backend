# Graph Report - backend  (2026-09-23)

## Corpus Check
- 130 files · ~172,985 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 5 file(s) not represented in the graph (top: (none) 4, .example 1)

## Summary
- 1020 nodes · 2880 edges · 50 communities (32 shown, 2 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 199 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `3d53d6c9`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- net/http.ResponseWriter
- notification_service_test.go
- AnnouncementWithFarm
- AuthService
- main
- testing.T
- Account
- .GetFarmStatistics
- Role
- field
- Plot
- time.Time
- NewRouter
- Schwarzes Brett (Farmer Announcement Board)
- Three-Layer Architecture Pattern
- Config
- OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)
- Queries
- email_sender_test.go
- github.com/google/uuid.UUID
- account.sql.go
- Service Layer (internal/services)
- github.com/twpayne/go-geom.Polygon
- GET/POST /api/rentals
- GetFarmByIDRow
- GET /api/statistics Role-derived Scope
- fakeFarmRepo
- Login Flow (argon2id + JWT)
- github.com/Neue-Konzepte-BaaS/backend
- .InsertBroadcastNotification
- context.Context
- Rental
- BroadcastNotification
- .FindPostalCodeCoordinates

## God Nodes (most connected - your core abstractions)
1. `main()` - 45 edges
2. `New()` - 35 edges
3. `Account` - 33 edges
4. `WriteError()` - 29 edges
5. `Recipient` - 29 edges
6. `setupTestDB()` - 28 edges
7. `WriteJSON()` - 26 edges
8. `Queries` - 26 edges
9. `NotificationService` - 25 edges
10. `seedCustomer()` - 24 edges

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

## Communities (50 total, 2 thin omitted)

### Community 0 - "net/http.ResponseWriter"
Cohesion: 0.08
Nodes (41): encoding/json.RawMessage, net/http.Request, net/http.ResponseWriter, AnnouncementHandler, announcementResponse, AuthHandler, createAnnouncementRequest, createAnnouncementResponse (+33 more)

### Community 1 - "notification_service_test.go"
Cohesion: 0.07
Nodes (48): html/template.Template, io/fs.FS, sync.Map, sync.Once, sync.WaitGroup, testing/fstest.MapFS, broadcastRequest, broadcastResponse (+40 more)

### Community 2 - "AnnouncementWithFarm"
Cohesion: 0.08
Nodes (27): createRipenessNoticeRequest, createRipenessNoticeResponse, ripenessNoticeResponse, toRipenessNoticeResponse(), AnnouncementWithFarm, RipenessNoticeWithDetails, toModelAnnouncement(), mapForeignKeyError() (+19 more)

### Community 3 - "AuthService"
Cohesion: 0.09
Nodes (30): HashPassword(), TestHashUsesFreshSalt(), TestHashVerify(), TestVerifyRejectsMalformed(), VerifyPassword(), Claims, Issuer, NewIssuer() (+22 more)

### Community 4 - "main"
Cohesion: 0.07
Nodes (52): main(), migrateDB(), seedAdmin(), FieldHandler, InboxHandler, inboxItemResponse, nearbyPlotResponse, PlotSearchHandler (+44 more)

### Community 5 - "testing.T"
Cohesion: 0.14
Nodes (55): DBTX, github.com/jackc/pgx/v5/pgxpool.Pool, testing.T, findAccount(), seedOrphanAccount(), TestGetAccountByEmail_IncludesPostalCode(), TestGetAccountByID_IncludesNameAndPostalCode(), TestListAccounts_CarriesNoPasswordMaterial() (+47 more)

### Community 6 - "Account"
Cohesion: 0.07
Nodes (8): Account, Recipient, isUniqueViolation(), accountRepository, fakeAccountListRepo, fakeAccountRepo, fakeRecipientRepo, fieldAndCrop

### Community 7 - ".GetFarmStatistics"
Cohesion: 0.40
Nodes (3): GetFarmStatisticsRow, GetPlatformStatisticsRow, Queries

### Community 8 - "Role"
Cohesion: 0.12
Nodes (22): time.Duration, AccountHandler, accountListingResponse, accountPageResponse, NewAccountHandler(), optionalRole(), toAccountPageResponse(), AccountListFilter (+14 more)

### Community 9 - "field"
Cohesion: 0.06
Nodes (34): account, admin, customer, farmer, idx_account_email, check_plot_within_field(), field, plot (+26 more)

### Community 10 - "Plot"
Cohesion: 0.09
Nodes (10): Field, FieldWithPlots, NearbyPlot, Plot, mapGeometryError(), fakeFieldRepo, fakePlotRepo, PlotWithCrops (+2 more)

### Community 11 - "time.Time"
Cohesion: 0.05
Nodes (51): ListFarmsRow, time.Time, accountStatisticsResponse, FarmHandler, farmListingResponse, farmOwnerResponse, farmPageResponse, farmResponse (+43 more)

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

### Community 17 - "Queries"
Cohesion: 0.20
Nodes (4): GetCropsByPlotsRow, InsertCropParams, InsertPlotCropParams, Queries

### Community 18 - "email_sender_test.go"
Cohesion: 0.13
Nodes (32): newEmailSender(), bytes.Buffer, mime/multipart.Writer, net/smtp.Auth, addressHeader(), encodeHeader(), needsEncoding(), NewConsoleEmailSender() (+24 more)

### Community 19 - "github.com/google/uuid.UUID"
Cohesion: 0.08
Nodes (39): Account, Admin, Announcement, BroadcastNotification, Crop, Customer, Farm, Farmer (+31 more)

### Community 20 - "account.sql.go"
Cohesion: 0.15
Nodes (10): GetAccountByEmailRow, GetAccountByIDRow, GetAllRecipientsRow, InsertAccountParams, InsertAdminParams, InsertCustomerParams, InsertFarmerParams, ListAccountsParams (+2 more)

### Community 21 - "Service Layer (internal/services)"
Cohesion: 0.23
Nodes (12): Dropped ST_Equals Rectangle CHECK Constraint, SQLSTATE-to-HTTP Error Translation Table, mapGeometryError(), trg_plot_within_field Trigger (ST_Within), AccountHandler (example), AccountRepository (example), AccountService (example), Repository-interface Dependency Inversion (+4 more)

### Community 22 - "github.com/twpayne/go-geom.Polygon"
Cohesion: 0.13
Nodes (12): GetFieldByIDRow, GetFieldsByFarmRow, GetNearestPlotsParams, GetNearestPlotsRow, GetPlotByIDRow, GetPlotsByFieldsRow, InsertFieldParams, InsertPlotParams (+4 more)

### Community 23 - "GET/POST /api/rentals"
Cohesion: 0.29
Nodes (8): Bauer as a Service (BaaS) Project, chi HTTP router, PostGIS + btree_gist extensions, Backend/Frontend Repo Scope Boundary, No Availability Check — Let Postgres Arbitrate, rental_no_overlap EXCLUDE Constraint (btree_gist), Rental / RentalWithPlot Schema, GET/POST /api/rentals

### Community 24 - "GetFarmByIDRow"
Cohesion: 0.22
Nodes (6): GetFarmByIDRow, InsertFarmParams, ListFarmsParams, github.com/jackc/pgx/v5/pgtype.Date, github.com/jackc/pgx/v5/pgtype.Int4, Queries

### Community 25 - "GET /api/statistics Role-derived Scope"
Cohesion: 0.40
Nodes (6): DISTINCT Audience Query via rental/plot/field Join, Single-round-trip Aggregate Query (MVCC snapshot consistency), GET /api/statistics Role-derived Scope, GET /api/statistics, Statistics Schema (farm/platform scope), GET /api/statistics (README description)

### Community 27 - "Login Flow (argon2id + JWT)"
Cohesion: 0.50
Nodes (5): argon2id Password Hashing, Login Flow (argon2id + JWT), HS256 JWT Access/Refresh Tokens, Timing-equaliser dummy hash verification, POST /api/auth/login

### Community 42 - ".InsertBroadcastNotification"
Cohesion: 0.32
Nodes (4): InsertBroadcastNotificationParams, Queries, broadcast_notification, idx_broadcast_notification_created

### Community 44 - "context.Context"
Cohesion: 0.12
Nodes (7): context.Context, Crop, toCrops(), checkFieldOwnership(), cropOffered(), cropRepository, fakeCropRepo

### Community 45 - "Rental"
Cohesion: 0.10
Nodes (19): fakeFarmRepo, fakeFieldRepo, fakePlotRepo, Rental, RentalStatus, RentalWithField, RentalWithPlot, RentalWithPlotAndCustomer (+11 more)

### Community 46 - "BroadcastNotification"
Cohesion: 0.33
Nodes (4): BroadcastNotification, toModelBroadcastNotification(), broadcastNotificationRepository, fakeBroadcastRepo

### Community 49 - ".FindPostalCodeCoordinates"
Cohesion: 0.50
Nodes (3): PostalCode, github.com/twpayne/go-geom.Point, Queries

## Knowledge Gaps
- **26 isolated node(s):** `createAnnouncementRequest`, `createCropRequest`, `loginRequest`, `registerRequest`, `rentPlotRequest` (+21 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 79 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `main` to `net/http.ResponseWriter`, `notification_service_test.go`, `AuthService`, `testing.T`, `Role`, `time.Time`, `NewRouter`, `Config`, `email_sender_test.go`?**
  _High betweenness centrality (0.082) - this node is a cross-community bridge._
- **Why does `MustClaimsFromContext()` connect `net/http.ResponseWriter` to `AuthService`, `NewRouter`, `context.Context`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **Why does `crop` connect `field` to `Queries`, `Plot`, `Rental`?**
  _High betweenness centrality (0.034) - this node is a cross-community bridge._
- **What connects `createAnnouncementRequest`, `createCropRequest`, `loginRequest` to the rest of the system?**
  _26 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.08031442241968557 - nodes in this community are weakly interconnected._
- **Should `notification_service_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.06746031746031746 - nodes in this community are weakly interconnected._
- **Should `AnnouncementWithFarm` be split into smaller, more focused modules?**
  _Cohesion score 0.07729468599033816 - nodes in this community are weakly interconnected._