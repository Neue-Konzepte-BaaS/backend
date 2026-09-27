# Graph Report - backend  (2026-09-27)

## Corpus Check
- 161 files · ~208,656 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 6 file(s) not represented in the graph (top: (none) 5, .example 1)

## Summary
- 1405 nodes · 4198 edges · 75 communities (50 shown, 4 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 250 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `f8f62077`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- net/http.ResponseWriter
- NotificationService
- AnnouncementWithFarm
- newTestService
- .GetPendingRegistrationByID
- testing.T
- Statistics
- rental.sql.go
- context.Context
- field
- Plot
- Page
- auth_test.go
- Schwarzes Brett (Farmer Announcement Board)
- Three-Layer Architecture Pattern
- InboxService
- OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)
- Queries
- email_sender_test.go
- github.com/google/uuid.UUID
- account.sql.go
- Service Layer (internal/services)
- github.com/twpayne/go-geom.Polygon
- GET/POST /api/rentals
- ListFarmsRow
- GET /api/statistics Role-derived Scope
- github.com/jackc/pgx/v5/pgtype.Timestamptz
- Login Flow (argon2id + JWT)
- 20260925071608_pending_registrations.sql
- github.com/Neue-Konzepte-BaaS/backend
- .InsertBroadcastNotification
- Crop
- Rental
- FarmRepository
- CareInstruction
- RentalCheckout
- .FindPostalCodeCoordinates
- care_instruction.sql.go
- Account
- plotRepository
- ComputeRentalPriceCents
- toModelAnnouncement
- Role
- rental_checkout
- AuthService
- .ListAccounts
- ripeness_notice.sql.go
- .GetFarmCropRates
- farmer
- newTestRentalService
- crop
- fakeNotificationService
- plot
- announcement

## God Nodes (most connected - your core abstractions)
1. `main()` - 53 edges
2. `New()` - 46 edges
3. `WriteError()` - 41 edges
4. `setupTestDB()` - 38 edges
5. `WriteJSON()` - 34 edges
6. `Account` - 33 edges
7. `Queries` - 32 edges
8. `Rental` - 30 edges
9. `Recipient` - 30 edges
10. `MustClaimsFromContext()` - 29 edges

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

## Communities (75 total, 4 thin omitted)

### Community 0 - "net/http.ResponseWriter"
Cohesion: 0.05
Nodes (65): encoding/json.RawMessage, net/http.Request, net/http.ResponseWriter, AnnouncementHandler, announcementResponse, AuthHandler, CareGuideHandler, careInstructionRequest (+57 more)

### Community 1 - "NotificationService"
Cohesion: 0.05
Nodes (55): html/template.Template, io/fs.FS, sync.Map, sync.Once, sync.WaitGroup, testing/fstest.MapFS, broadcastRequest, broadcastResponse (+47 more)

### Community 2 - "AnnouncementWithFarm"
Cohesion: 0.15
Nodes (18): AnnouncementWithFarm, AnnouncementService, NewAnnouncementService(), newTestAnnouncementService(), TestCreateAnnouncement_BothFieldAndPlotIsRejected(), TestCreateAnnouncement_MailFailureStillKeepsTheAnnouncement(), TestCreateAnnouncement_ScopedToOwnField_PassesScopeThrough(), TestCreateAnnouncement_ScopedToOwnPlot_ResolvesOwnershipThroughItsField() (+10 more)

### Community 3 - "newTestService"
Cohesion: 0.17
Nodes (26): HashPassword(), TestHashUsesFreshSalt(), TestHashVerify(), TestVerifyRejectsMalformed(), VerifyPassword(), RegisterInput, newFakePendingRegistrationRepo(), newTestService() (+18 more)

### Community 4 - ".GetPendingRegistrationByID"
Cohesion: 0.33
Nodes (3): GetPendingRegistrationByIDRow, UpsertPendingRegistrationParams, Queries

### Community 5 - "testing.T"
Cohesion: 0.06
Nodes (113): main(), migrateDB(), seedAdmin(), DBTX, github.com/jackc/pgx/v5/pgxpool.Pool, testing.T, Config, Load() (+105 more)

### Community 6 - "Statistics"
Cohesion: 0.10
Nodes (26): accountStatisticsResponse, fieldStatisticsResponse, plotStatisticsResponse, rentalStatisticsResponse, StatisticsHandler, statisticsResponse, NewStatisticsHandler(), toStatisticsResponse() (+18 more)

### Community 7 - "rental.sql.go"
Cohesion: 0.15
Nodes (11): GetActiveRentalsByCustomerRow, GetRentalByIDRow, GetRentalsByCustomerRow, GetRentalsByFarmRow, GetRentalWithFieldByIDRow, InsertRentalRequestParams, InsertRentalRequestRow, IsPlotAvailableParams (+3 more)

### Community 8 - "context.Context"
Cohesion: 0.12
Nodes (5): context.Context, Recipient, fakeAccountListRepo, fakeRecipientRepo, fieldAndCrop

### Community 9 - "field"
Cohesion: 0.19
Nodes (10): check_plot_within_field(), field, check_plot_within_field, trg_plot_within_field, check_plot_within_field(), check_plot_within_field, trg_plot_within_field, farm (+2 more)

### Community 10 - "Plot"
Cohesion: 0.12
Nodes (7): FieldWithPlots, NearbyPlot, Plot, PlotCropOffering, PlotWithCrops, fakePaymentPlotRepo, fakePlotRepo

### Community 11 - "Page"
Cohesion: 0.06
Nodes (35): github.com/Neue-Konzepte-BaaS/backend/internal/models.FarmUpdate, farmCropRateResponse, farmCropRatesResponse, farmListingResponse, farmOwnerResponse, farmPageResponse, farmResponse, setFarmCropRatesRequest (+27 more)

### Community 12 - "auth_test.go"
Cohesion: 0.13
Nodes (29): net/http.Cookie, net/http.Handler, net/http/httptest.ResponseRecorder, ClaimsFromContext(), RequireAuth(), assertSameClaims(), assertUnauthorized(), newRequest() (+21 more)

### Community 13 - "Schwarzes Brett (Farmer Announcement Board)"
Cohesion: 0.13
Nodes (19): POST /api/announcements (Schwarzes Brett), services.Dispatcher, EmailSender (internal/repositories/email_sender.go), services.NotificationService, POST /api/notifications, Best-effort Async Delivery After Response, Board-not-Inbox Visibility Design Choice, Embedded Templates (embed.FS) Rationale (+11 more)

### Community 14 - "Three-Layer Architecture Pattern"
Cohesion: 0.12
Nodes (19): config.Load(), testcontainers integration testing, BaaS Backend Architecture, Containerfile Multi-stage Deployment, Known Gaps and Rough Edges (12 items), sqlc + dbmate Migrations-as-schema-of-record, Applying Three-Layer Pattern in Microservices, Three-Layer Architecture Pattern (+11 more)

### Community 15 - "InboxService"
Cohesion: 0.29
Nodes (8): InboxHandler, inboxItemResponse, NewInboxHandler(), toInboxItemResponse(), InboxItem, InboxItemKind, careInboxItems(), InboxService

### Community 16 - "OpenAPI 3.1 Spec (Neue Konzepte BaaS Backend API)"
Cohesion: 0.18
Nodes (15): PostGIS KNN Operator <-> with GiST Index, Role Derivation via Subtype Table Join, GET /api/plots/nearest Spatial Search, POST /api/auth/logout, GET /api/auth/me, POST /api/auth/register, Crop Schema, GET/POST /api/crops (+7 more)

### Community 17 - "Queries"
Cohesion: 0.15
Nodes (8): GetAllCropsRow, GetCropByIDRow, GetCropsByPlotRow, GetCropsByPlotsRow, GetPricedCropOfferingsByPlotsRow, InsertCropParams, InsertPlotCropParams, Queries

### Community 18 - "email_sender_test.go"
Cohesion: 0.13
Nodes (32): newEmailSender(), bytes.Buffer, mime/multipart.Writer, net/smtp.Auth, addressHeader(), encodeHeader(), needsEncoding(), NewConsoleEmailSender() (+24 more)

### Community 19 - "github.com/google/uuid.UUID"
Cohesion: 0.11
Nodes (16): CompleteRentalCheckoutParams, GetAnnouncementsByFarmerRow, GetAnnouncementsForCustomerRow, GetCustomersOfFarmerForFieldRow, GetCustomersOfFarmerForPlotRow, GetCustomersOfFarmerRow, InsertAnnouncementParams, InsertAnnouncementRow (+8 more)

### Community 20 - "account.sql.go"
Cohesion: 0.15
Nodes (10): GetAccountByEmailRow, GetAccountByIDRow, GetAllRecipientsRow, InsertAccountParams, InsertAdminParams, InsertCustomerParams, InsertFarmerParams, ListAccountsParams (+2 more)

### Community 21 - "Service Layer (internal/services)"
Cohesion: 0.23
Nodes (12): Dropped ST_Equals Rectangle CHECK Constraint, SQLSTATE-to-HTTP Error Translation Table, mapGeometryError(), trg_plot_within_field Trigger (ST_Within), AccountHandler (example), AccountRepository (example), AccountService (example), Repository-interface Dependency Inversion (+4 more)

### Community 22 - "github.com/twpayne/go-geom.Polygon"
Cohesion: 0.10
Nodes (16): Field, GetFieldByIDRow, GetFieldsByFarmRow, GetNearestPlotsParams, GetNearestPlotsRow, GetPlotByIDRow, GetPlotsByFieldsRow, InsertFieldParams (+8 more)

### Community 23 - "GET/POST /api/rentals"
Cohesion: 0.29
Nodes (8): Bauer as a Service (BaaS) Project, chi HTTP router, PostGIS + btree_gist extensions, Backend/Frontend Repo Scope Boundary, No Availability Check — Let Postgres Arbitrate, rental_no_overlap EXCLUDE Constraint (btree_gist), Rental / RentalWithPlot Schema, GET/POST /api/rentals

### Community 24 - "ListFarmsRow"
Cohesion: 0.21
Nodes (7): Farm, GetFarmByIDRow, InsertFarmParams, ListFarmsParams, ListFarmsRow, github.com/jackc/pgx/v5/pgtype.Date, Queries

### Community 25 - "GET /api/statistics Role-derived Scope"
Cohesion: 0.40
Nodes (6): DISTINCT Audience Query via rental/plot/field Join, Single-round-trip Aggregate Query (MVCC snapshot consistency), GET /api/statistics Role-derived Scope, GET /api/statistics, Statistics Schema (farm/platform scope), GET /api/statistics (README description)

### Community 26 - "github.com/jackc/pgx/v5/pgtype.Timestamptz"
Cohesion: 0.12
Nodes (20): Account, Admin, Announcement, BroadcastNotification, CareInstruction, Crop, Customer, FarmCareGuide (+12 more)

### Community 27 - "Login Flow (argon2id + JWT)"
Cohesion: 0.50
Nodes (5): argon2id Password Hashing, Login Flow (argon2id + JWT), HS256 JWT Access/Refresh Tokens, Timing-equaliser dummy hash verification, POST /api/auth/login

### Community 42 - ".InsertBroadcastNotification"
Cohesion: 0.32
Nodes (4): InsertBroadcastNotificationParams, Queries, broadcast_notification, idx_broadcast_notification_created

### Community 44 - "Crop"
Cohesion: 0.10
Nodes (5): Crop, isUniqueViolation(), cropRepository, fakeCropRepo, fakePaymentCropRepo

### Community 45 - "Rental"
Cohesion: 0.06
Nodes (31): time.Time, ActiveRental, Rental, RentalStatus, RentalWithField, RentalWithPlot, RentalWithPlotAndCustomer, mapRentalError() (+23 more)

### Community 46 - "FarmRepository"
Cohesion: 0.06
Nodes (44): createRipenessNoticeRequest, createRipenessNoticeResponse, RipenessNoticeHandler, ripenessNoticeResponse, NewFieldHandler(), NewRentalHandler(), NewRipenessNoticeHandler(), toRipenessNoticeResponse() (+36 more)

### Community 47 - "CareInstruction"
Cohesion: 0.08
Nodes (28): careInstructionRow, CareInstruction, CropAtFarm, CropCareGuide, PlotCareGuide, mapCareInstructionError(), toCareInstruction(), mapForeignKeyError() (+20 more)

### Community 48 - "RentalCheckout"
Cohesion: 0.18
Nodes (7): CheckoutSessionStatusResult, RentalCheckout, toModelRentalCheckout(), sessionIDFor(), CheckoutStatus, rentalCheckoutRepository, fakeCheckoutRepo

### Community 49 - ".FindPostalCodeCoordinates"
Cohesion: 0.50
Nodes (3): PostalCode, github.com/twpayne/go-geom.Point, Queries

### Community 50 - "care_instruction.sql.go"
Cohesion: 0.11
Nodes (15): CopyDefaultCareInstructionsToFarmParams, DeleteFarmCareGuideParams, GetCareInstructionByIDRow, GetDefaultCareInstructionsByCropRow, GetEffectiveCareInstructionsParams, GetEffectiveCareInstructionsRow, GetFarmCopyOfCareInstructionParams, GetFarmCopyOfCareInstructionRow (+7 more)

### Community 53 - "Account"
Cohesion: 0.23
Nodes (3): Account, accountRepository, fakeAccountRepo

### Community 55 - "plotRepository"
Cohesion: 0.24
Nodes (3): mapGeometryError(), fromPgInt4(), plotRepository

### Community 56 - "ComputeRentalPriceCents"
Cohesion: 0.40
Nodes (3): ComputeRentalPriceCents(), TestComputeRentalPriceCents(), TestComputeRentalPriceCents_MatchesTheDocumentedFormula()

### Community 58 - "Role"
Cohesion: 0.16
Nodes (9): time.Duration, AccountListFilter, AccountListing, PendingRegistration, Role, toModelAccountListing(), clampPagination(), pendingRegistrationRepository (+1 more)

### Community 60 - "rental_checkout"
Cohesion: 0.26
Nodes (5): Queries, farm_crop_rate, idx_rental_checkout_customer, idx_rental_checkout_rental, rental_checkout

### Community 62 - "AuthService"
Cohesion: 0.13
Nodes (10): Claims, Issuer, NewPendingRegistrationRepository(), AuthService, TokenPair, NewAuthService(), PendingRegistrationRepository, jwt.RegisteredClaims (+2 more)

### Community 63 - ".ListAccounts"
Cohesion: 0.22
Nodes (14): AccountHandler, accountListingResponse, accountPageResponse, NewAccountHandler(), optionalRole(), toAccountPageResponse(), AccountService, NewAccountService() (+6 more)

### Community 64 - "ripeness_notice.sql.go"
Cohesion: 0.29
Nodes (6): GetCustomersOfFarmerForFieldAndCropParams, GetCustomersOfFarmerForFieldAndCropRow, GetRipenessNoticesForCustomerRow, InsertRipenessNoticeParams, InsertRipenessNoticeRow, Queries

### Community 66 - ".GetFarmCropRates"
Cohesion: 0.28
Nodes (4): GetFarmCropRateParams, GetFarmCropRatesRow, InsertFarmCropRateParams, Queries

### Community 68 - "farmer"
Cohesion: 0.36
Nodes (7): account, admin, customer, farmer, idx_account_email, idx_ripeness_notice_field_created, ripeness_notice

### Community 70 - "newTestRentalService"
Cohesion: 0.46
Nodes (7): newTestRentalService(), TestRentalService_ApproveRental_ForbiddenWhenFarmerDoesNotOwnPlot(), TestRentalService_ApproveRental_OK(), TestRentalService_DeclineRental_AlreadyDecided(), TestRentalService_RequestRental_OK(), TestRentalService_RequestRental_RejectsBlankMessage(), TestRentalService_RequestRental_RejectsStartDateOutsideWindow()

### Community 72 - "crop"
Cohesion: 0.23
Nodes (10): crop, field_crop, field_crop, plot_crop, care_instruction, idx_care_instruction_crop_week, farm_care_guide, idx_care_instruction_crop_farm_week (+2 more)

### Community 75 - "plot"
Cohesion: 0.21
Nodes (10): plot, idx_plot_coordinates, idx_postal_code_coordinates, idx_postal_code_name, idx_postal_code_zipcode, postal_code, idx_rental_customer, rental (+2 more)

### Community 76 - "announcement"
Cohesion: 0.50
Nodes (3): announcement, idx_announcement_farmer_created, idx_announcement_plot

## Knowledge Gaps
- **30 isolated node(s):** `checkoutSessionResponse`, `createAnnouncementRequest`, `createCheckoutSessionRequest`, `createCropRequest`, `loginRequest` (+25 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 91 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `MustClaimsFromContext()` connect `net/http.ResponseWriter` to `context.Context`, `auth_test.go`, `AuthService`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Why does `main()` connect `testing.T` to `net/http.ResponseWriter`, `NotificationService`, `AnnouncementWithFarm`, `Statistics`, `Page`, `FarmRepository`, `InboxService`, `CareInstruction`, `email_sender_test.go`, `AuthService`, `.ListAccounts`?**
  _High betweenness centrality (0.041) - this node is a cross-community bridge._
- **Why does `plot` connect `plot` to `crop`, `field`, `rental_checkout`, `Rental`?**
  _High betweenness centrality (0.028) - this node is a cross-community bridge._
- **What connects `checkoutSessionResponse`, `createAnnouncementRequest`, `createCheckoutSessionRequest` to the rest of the system?**
  _30 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.05472263868065967 - nodes in this community are weakly interconnected._
- **Should `NotificationService` be split into smaller, more focused modules?**
  _Cohesion score 0.053613053613053616 - nodes in this community are weakly interconnected._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.05661881977671451 - nodes in this community are weakly interconnected._