# Graph Report - backend  (2026-09-27)

## Corpus Check
- 161 files · ~208,753 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 6 file(s) not represented in the graph (top: (none) 5, .example 1)

## Summary
- 1407 nodes · 4202 edges · 83 communities (58 shown, 4 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 250 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `7ed952d9`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- net/http.ResponseWriter
- NotificationService
- announcement_service_test.go
- newTestService
- .GetPendingRegistrationByID
- testing.T
- Statistics
- rental.sql.go
- context.Context
- field
- Plot
- fakeFarmRepo
- auth_test.go
- Schwarzes Brett (Farmer Announcement Board)
- Three-Layer Architecture Pattern
- NewInboxService
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
- payment_service_test.go
- .FindPostalCodeCoordinates
- care_instruction.sql.go
- announcement.sql.go
- main
- NewFarmService
- AnnouncementWithFarm
- Page
- FarmListing
- rental_checkout
- RipenessNoticeService
- Claims
- .ListAccounts
- ripeness_notice.sql.go
- AnnouncementService
- .GetFarmCropRates
- account_handler.go
- farmer
- crop
- newTestRentalService
- PlotSearchService
- care_instruction
- fakeNotificationService
- ActiveRental
- 20260913114701_postal_codes.sql
- announcement
- 20260915135248_drop_plot_check.sql

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

## Communities (83 total, 4 thin omitted)

### Community 0 - "net/http.ResponseWriter"
Cohesion: 0.06
Nodes (62): encoding/json.RawMessage, net/http.Request, net/http.ResponseWriter, AnnouncementHandler, announcementResponse, AuthHandler, CareGuideHandler, careInstructionRequest (+54 more)

### Community 1 - "NotificationService"
Cohesion: 0.07
Nodes (48): html/template.Template, io/fs.FS, sync.Map, sync.Once, sync.WaitGroup, testing/fstest.MapFS, broadcastRequest, broadcastResponse (+40 more)

### Community 2 - "announcement_service_test.go"
Cohesion: 0.27
Nodes (14): NewAnnouncementService(), newTestAnnouncementService(), TestCreateAnnouncement_BothFieldAndPlotIsRejected(), TestCreateAnnouncement_MailFailureStillKeepsTheAnnouncement(), TestCreateAnnouncement_ScopedToOwnField_PassesScopeThrough(), TestCreateAnnouncement_ScopedToOwnPlot_ResolvesOwnershipThroughItsField(), TestCreateAnnouncement_ScopeFieldNotFound(), TestCreateAnnouncement_ScopeOwnedByAnotherFarmerIsForbidden() (+6 more)

### Community 3 - "newTestService"
Cohesion: 0.07
Nodes (42): time.Duration, HashPassword(), TestHashUsesFreshSalt(), TestHashVerify(), TestVerifyRejectsMalformed(), VerifyPassword(), Issuer, NewIssuer() (+34 more)

### Community 4 - ".GetPendingRegistrationByID"
Cohesion: 0.33
Nodes (3): GetPendingRegistrationByIDRow, UpsertPendingRegistrationParams, Queries

### Community 5 - "testing.T"
Cohesion: 0.07
Nodes (93): DBTX, github.com/jackc/pgx/v5/pgxpool.Pool, testing.T, Config, Load(), parseBoolEnv(), parseIntEnv(), TestParseIntEnv() (+85 more)

### Community 6 - "Statistics"
Cohesion: 0.08
Nodes (35): accountStatisticsResponse, farmCropRateResponse, farmCropRatesResponse, farmListingResponse, farmOwnerResponse, farmPageResponse, farmResponse, fieldStatisticsResponse (+27 more)

### Community 7 - "rental.sql.go"
Cohesion: 0.15
Nodes (11): GetActiveRentalsByCustomerRow, GetRentalByIDRow, GetRentalsByCustomerRow, GetRentalsByFarmRow, GetRentalWithFieldByIDRow, InsertRentalRequestParams, InsertRentalRequestRow, IsPlotAvailableParams (+3 more)

### Community 8 - "context.Context"
Cohesion: 0.10
Nodes (10): context.Context, Account, Role, Recipient, isUniqueViolation(), accountRepository, fakeAccountListRepo, fakeAccountRepo (+2 more)

### Community 9 - "field"
Cohesion: 0.20
Nodes (11): check_plot_within_field(), field, plot, check_plot_within_field, trg_plot_within_field, idx_field_farmer, idx_plot_field, farm (+3 more)

### Community 10 - "Plot"
Cohesion: 0.11
Nodes (8): FieldWithPlots, NearbyPlot, Plot, PlotWithCrops, mapGeometryError(), fromPgInt4(), plotRepository, fakePaymentPlotRepo

### Community 11 - "fakeFarmRepo"
Cohesion: 0.11
Nodes (7): github.com/Neue-Konzepte-BaaS/backend/internal/models.FarmUpdate, NewFarmHandler(), FarmCropRate, Farm, FarmService, fakeFarmRepo, fakePaymentFarmRepo

### Community 12 - "auth_test.go"
Cohesion: 0.13
Nodes (29): net/http.Cookie, net/http.Handler, net/http/httptest.ResponseRecorder, ClaimsFromContext(), RequireAuth(), assertSameClaims(), assertUnauthorized(), newRequest() (+21 more)

### Community 13 - "Schwarzes Brett (Farmer Announcement Board)"
Cohesion: 0.13
Nodes (19): POST /api/announcements (Schwarzes Brett), services.Dispatcher, EmailSender (internal/repositories/email_sender.go), services.NotificationService, POST /api/notifications, Best-effort Async Delivery After Response, Board-not-Inbox Visibility Design Choice, Embedded Templates (embed.FS) Rationale (+11 more)

### Community 14 - "Three-Layer Architecture Pattern"
Cohesion: 0.12
Nodes (19): config.Load(), testcontainers integration testing, BaaS Backend Architecture, Containerfile Multi-stage Deployment, Known Gaps and Rough Edges (12 items), sqlc + dbmate Migrations-as-schema-of-record, Applying Three-Layer Pattern in Microservices, Three-Layer Architecture Pattern (+11 more)

### Community 15 - "NewInboxService"
Cohesion: 0.08
Nodes (28): InboxHandler, inboxItemResponse, NewInboxHandler(), toInboxItemResponse(), InboxItem, InboxItemKind, BroadcastNotification, RipenessNoticeWithDetails (+20 more)

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
Cohesion: 0.15
Nodes (9): CompleteRentalCheckoutParams, InsertRentalCheckoutParams, github.com/google/uuid.UUID, Field, fieldRepository, fakeFieldRepo, fakeNotifier, fakePaymentFieldRepo (+1 more)

### Community 20 - "account.sql.go"
Cohesion: 0.16
Nodes (9): GetAccountByEmailRow, GetAccountByIDRow, GetAllRecipientsRow, InsertAccountParams, InsertAdminParams, InsertCustomerParams, InsertFarmerParams, ListAccountsParams (+1 more)

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
Cohesion: 0.11
Nodes (21): Account, Admin, Announcement, BroadcastNotification, CareInstruction, Crop, Customer, FarmCareGuide (+13 more)

### Community 27 - "Login Flow (argon2id + JWT)"
Cohesion: 0.50
Nodes (5): argon2id Password Hashing, Login Flow (argon2id + JWT), HS256 JWT Access/Refresh Tokens, Timing-equaliser dummy hash verification, POST /api/auth/login

### Community 42 - ".InsertBroadcastNotification"
Cohesion: 0.32
Nodes (4): InsertBroadcastNotificationParams, Queries, broadcast_notification, idx_broadcast_notification_created

### Community 44 - "Crop"
Cohesion: 0.09
Nodes (6): Crop, PlotCropOffering, toCrops(), cropRepository, fakeCropRepo, fakePaymentCropRepo

### Community 45 - "Rental"
Cohesion: 0.09
Nodes (13): time.Time, CheckoutSessionResult, Rental, RentalStatus, RentalWithField, RentalWithPlot, RentalWithPlotAndCustomer, mapRentalError() (+5 more)

### Community 46 - "FarmRepository"
Cohesion: 0.21
Nodes (18): NewFieldHandler(), NewPaymentHandler(), NewRentalHandler(), CropService, NewCropService(), FieldService, PlotService, NewFieldService() (+10 more)

### Community 47 - "CareInstruction"
Cohesion: 0.08
Nodes (26): careInstructionRow, CareInstruction, CropAtFarm, CropCareGuide, PlotCareGuide, mapCareInstructionError(), toCareInstruction(), mapForeignKeyError() (+18 more)

### Community 48 - "payment_service_test.go"
Cohesion: 0.08
Nodes (28): CheckoutSessionStatusResult, RentalCheckout, toModelRentalCheckout(), int32Ptr(), newTestPaymentService(), sessionIDFor(), TestCreateCheckoutSession_ComputesPriceAndRecordsAPendingCheckout(), TestCreateCheckoutSession_CropNotOfferedNeverReachesStripe() (+20 more)

### Community 49 - ".FindPostalCodeCoordinates"
Cohesion: 0.50
Nodes (3): PostalCode, github.com/twpayne/go-geom.Point, Queries

### Community 50 - "care_instruction.sql.go"
Cohesion: 0.11
Nodes (15): CopyDefaultCareInstructionsToFarmParams, DeleteFarmCareGuideParams, GetCareInstructionByIDRow, GetDefaultCareInstructionsByCropRow, GetEffectiveCareInstructionsParams, GetEffectiveCareInstructionsRow, GetFarmCopyOfCareInstructionParams, GetFarmCopyOfCareInstructionRow (+7 more)

### Community 53 - "announcement.sql.go"
Cohesion: 0.19
Nodes (8): GetAnnouncementsByFarmerRow, GetAnnouncementsForCustomerRow, GetCustomersOfFarmerForFieldRow, GetCustomersOfFarmerForPlotRow, GetCustomersOfFarmerRow, InsertAnnouncementParams, InsertAnnouncementRow, Queries

### Community 55 - "main"
Cohesion: 0.13
Nodes (15): main(), migrateDB(), seedAdmin(), NewCropHandler(), NewRentalCheckoutRepository(), NewStripeGateway(), NewCareGuideService(), AccountRepository (+7 more)

### Community 56 - "NewFarmService"
Cohesion: 0.30
Nodes (13): NewFarmService(), TestFarmService_GetFarm_NotFoundPropagates(), TestFarmService_GetFarm_OK(), TestGetCropRates_ResolvesTheCallersOwnFarm(), TestListFarms_ClampsPaginationAndTrimsTheSearchTerm(), TestListFarms_DerivesOccupancyTheSameWayAsStatistics(), TestListFarms_DerivesPlotFiguresPerFarm(), TestListFarms_FarmerCannotListFarms() (+5 more)

### Community 57 - "AnnouncementWithFarm"
Cohesion: 0.29
Nodes (5): AnnouncementWithFarm, toModelAnnouncement(), Announcement, announcementRepository, fakeInboxAnnouncementRepo

### Community 58 - "Page"
Cohesion: 0.26
Nodes (6): AccountListFilter, AccountListing, Page, toModelAccountListing(), clampPagination(), T

### Community 59 - "FarmListing"
Cohesion: 0.23
Nodes (4): FarmListFilter, FarmListing, toModelFarmListing(), farmRepository

### Community 60 - "rental_checkout"
Cohesion: 0.29
Nodes (4): Queries, idx_rental_checkout_customer, idx_rental_checkout_rental, rental_checkout

### Community 61 - "RipenessNoticeService"
Cohesion: 0.29
Nodes (9): createRipenessNoticeRequest, createRipenessNoticeResponse, RipenessNoticeHandler, ripenessNoticeResponse, NewRipenessNoticeHandler(), toRipenessNoticeResponse(), RipenessNoticeRepository, RipenessNoticeService (+1 more)

### Community 62 - "Claims"
Cohesion: 0.25
Nodes (3): Claims, jwt.RegisteredClaims, stubAuthService

### Community 63 - ".ListAccounts"
Cohesion: 0.50
Nodes (7): NewAccountService(), TestListAccounts_ClampsPagination(), TestListAccounts_OnlyAdminReachesTheRepository(), TestListAccounts_PassesThePageThrough(), TestListAccounts_RejectsUnknownRoleFilter(), TestListAccounts_TrimsTheSearchTerm(), TestListAccounts_WrapsRepositoryErrors()

### Community 64 - "ripeness_notice.sql.go"
Cohesion: 0.29
Nodes (6): GetCustomersOfFarmerForFieldAndCropParams, GetCustomersOfFarmerForFieldAndCropRow, GetRipenessNoticesForCustomerRow, InsertRipenessNoticeParams, InsertRipenessNoticeRow, Queries

### Community 65 - "AnnouncementService"
Cohesion: 0.22
Nodes (3): NewAnnouncementHandler(), AnnouncementService, checkFieldOwnership()

### Community 66 - ".GetFarmCropRates"
Cohesion: 0.28
Nodes (4): GetFarmCropRateParams, GetFarmCropRatesRow, InsertFarmCropRateParams, Queries

### Community 67 - "account_handler.go"
Cohesion: 0.36
Nodes (7): AccountHandler, accountListingResponse, accountPageResponse, NewAccountHandler(), optionalRole(), toAccountPageResponse(), AccountService

### Community 68 - "farmer"
Cohesion: 0.36
Nodes (7): account, admin, customer, farmer, idx_account_email, idx_rental_customer, rental

### Community 69 - "crop"
Cohesion: 0.31
Nodes (6): crop, field_crop, field_crop, plot_crop, idx_ripeness_notice_field_created, ripeness_notice

### Community 70 - "newTestRentalService"
Cohesion: 0.46
Nodes (7): newTestRentalService(), TestRentalService_ApproveRental_ForbiddenWhenFarmerDoesNotOwnPlot(), TestRentalService_ApproveRental_OK(), TestRentalService_DeclineRental_AlreadyDecided(), TestRentalService_RequestRental_OK(), TestRentalService_RequestRental_RejectsBlankMessage(), TestRentalService_RequestRental_RejectsStartDateOutsideWindow()

### Community 71 - "PlotSearchService"
Cohesion: 0.32
Nodes (5): NewPostalCodeRepository(), PlotSearchService, NewPlotSearchService(), PostalCodeRepository, postalCodeRepository

### Community 72 - "care_instruction"
Cohesion: 0.43
Nodes (6): care_instruction, idx_care_instruction_crop_week, farm_care_guide, idx_care_instruction_crop_farm_week, idx_care_instruction_crop_week, idx_care_instruction_farm_based_on

### Community 75 - "20260913114701_postal_codes.sql"
Cohesion: 0.53
Nodes (5): idx_plot_coordinates, idx_postal_code_coordinates, idx_postal_code_name, idx_postal_code_zipcode, postal_code

### Community 76 - "announcement"
Cohesion: 0.50
Nodes (3): announcement, idx_announcement_farmer_created, idx_announcement_plot

### Community 82 - "20260915135248_drop_plot_check.sql"
Cohesion: 0.50
Nodes (3): check_plot_within_field(), check_plot_within_field, trg_plot_within_field

## Knowledge Gaps
- **30 isolated node(s):** `checkoutSessionResponse`, `createAnnouncementRequest`, `createCheckoutSessionRequest`, `createCropRequest`, `loginRequest` (+25 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 92 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `main` to `net/http.ResponseWriter`, `AnnouncementService`, `NotificationService`, `account_handler.go`, `newTestService`, `testing.T`, `Statistics`, `PlotSearchService`, `announcement_service_test.go`, `fakeFarmRepo`, `FarmRepository`, `NewInboxService`, `email_sender_test.go`, `NewFarmService`, `RipenessNoticeService`, `.ListAccounts`?**
  _High betweenness centrality (0.047) - this node is a cross-community bridge._
- **Why does `MustClaimsFromContext()` connect `net/http.ResponseWriter` to `context.Context`, `auth_test.go`, `Claims`?**
  _High betweenness centrality (0.047) - this node is a cross-community bridge._
- **Why does `crop` connect `crop` to `care_instruction`, `field`, `Plot`, `ActiveRental`, `Crop`, `Rental`, `CareInstruction`, `rental_checkout`?**
  _High betweenness centrality (0.043) - this node is a cross-community bridge._
- **What connects `checkoutSessionResponse`, `createAnnouncementRequest`, `createCheckoutSessionRequest` to the rest of the system?**
  _30 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.056731984829329965 - nodes in this community are weakly interconnected._
- **Should `NotificationService` be split into smaller, more focused modules?**
  _Cohesion score 0.06746031746031746 - nodes in this community are weakly interconnected._
- **Should `newTestService` be split into smaller, more focused modules?**
  _Cohesion score 0.06927551560021153 - nodes in this community are weakly interconnected._