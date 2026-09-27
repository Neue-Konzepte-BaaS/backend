# Graph Report - backend  (2026-09-27)

## Corpus Check
- 161 files · ~206,712 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 6 file(s) not represented in the graph (top: (none) 5, .example 1)

## Summary
- 1402 nodes · 4192 edges · 82 communities (56 shown, 5 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 248 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `f22ab9c3`
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
- crop
- Plot
- NewFarmService
- auth_test.go
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
- ListFarmsRow
- GET /api/statistics Role-derived Scope
- github.com/jackc/pgx/v5/pgtype.Timestamptz
- Login Flow (argon2id + JWT)
- 20260925071608_pending_registrations.sql
- github.com/Neue-Konzepte-BaaS/backend
- .InsertBroadcastNotification
- Crop
- Rental
- main
- CareInstruction
- payment_service_test.go
- .FindPostalCodeCoordinates
- care_instruction.sql.go
- announcement.sql.go
- Role
- accountRepository
- farmer
- fakePendingRegistrationRepo
- Claims
- rental_checkout
- RentalStatus
- AuthService
- ActiveRental
- ripeness_notice.sql.go
- RipenessNoticeWithDetails
- .GetFarmCropRates
- account_handler.go
- PlotCropOffering
- .ListAccounts
- newTestRentalService
- care_instruction
- HashPassword
- fakeNotificationService
- RentalWithPlot
- 20260913114701_postal_codes.sql
- 20260915135248_drop_plot_check.sql

## God Nodes (most connected - your core abstractions)
1. `main()` - 53 edges
2. `New()` - 45 edges
3. `WriteError()` - 41 edges
4. `setupTestDB()` - 37 edges
5. `WriteJSON()` - 34 edges
6. `Account` - 33 edges
7. `Queries` - 32 edges
8. `Recipient` - 30 edges
9. `Rental` - 30 edges
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

## Communities (82 total, 5 thin omitted)

### Community 0 - "net/http.ResponseWriter"
Cohesion: 0.06
Nodes (63): encoding/json.RawMessage, net/http.Request, net/http.ResponseWriter, AnnouncementHandler, announcementResponse, AuthHandler, CareGuideHandler, careInstructionRequest (+55 more)

### Community 1 - "NotificationService"
Cohesion: 0.05
Nodes (55): html/template.Template, io/fs.FS, sync.Map, sync.Once, sync.WaitGroup, testing/fstest.MapFS, broadcastRequest, broadcastResponse (+47 more)

### Community 2 - "AnnouncementWithFarm"
Cohesion: 0.13
Nodes (20): AnnouncementWithFarm, toModelAnnouncement(), AnnouncementService, NewAnnouncementService(), newTestAnnouncementService(), TestCreateAnnouncement_BothFieldAndPlotIsRejected(), TestCreateAnnouncement_MailFailureStillKeepsTheAnnouncement(), TestCreateAnnouncement_ScopedToOwnField_PassesScopeThrough() (+12 more)

### Community 3 - "newTestService"
Cohesion: 0.28
Nodes (20): newFakePendingRegistrationRepo(), newTestService(), registerAndWait(), TestMe_NotFound(), TestMe_OK(), TestRegister_Customer(), TestRegister_EmailTakenPropagates(), TestRegister_Farmer_OK() (+12 more)

### Community 4 - ".GetPendingRegistrationByID"
Cohesion: 0.33
Nodes (3): GetPendingRegistrationByIDRow, UpsertPendingRegistrationParams, Queries

### Community 5 - "testing.T"
Cohesion: 0.09
Nodes (81): DBTX, github.com/jackc/pgx/v5/pgxpool.Pool, testing.T, findAccount(), seedOrphanAccount(), TestGetAccountByEmail_IncludesPostalCode(), TestGetAccountByID_IncludesNameAndPostalCode(), TestListAccounts_CarriesNoPasswordMaterial() (+73 more)

### Community 6 - "Statistics"
Cohesion: 0.07
Nodes (36): accountStatisticsResponse, farmCropRateResponse, farmCropRatesResponse, farmListingResponse, farmOwnerResponse, farmPageResponse, farmResponse, fieldStatisticsResponse (+28 more)

### Community 7 - "rental.sql.go"
Cohesion: 0.15
Nodes (11): GetActiveRentalsByCustomerRow, GetRentalByIDRow, GetRentalsByCustomerRow, GetRentalsByFarmRow, GetRentalWithFieldByIDRow, InsertRentalRequestParams, InsertRentalRequestRow, IsPlotAvailableParams (+3 more)

### Community 8 - "context.Context"
Cohesion: 0.12
Nodes (8): context.Context, Account, Recipient, stubAuthService, fakeAccountListRepo, fakeAccountRepo, fakeRecipientRepo, fieldAndCrop

### Community 9 - "crop"
Cohesion: 0.13
Nodes (17): check_plot_within_field(), field, plot, check_plot_within_field, trg_plot_within_field, idx_plot_coordinates, crop, field_crop (+9 more)

### Community 10 - "Plot"
Cohesion: 0.11
Nodes (8): FieldWithPlots, NearbyPlot, Plot, PlotWithCrops, fromPgInt4(), plotRepository, fakePaymentPlotRepo, fakePlotRepo

### Community 11 - "NewFarmService"
Cohesion: 0.07
Nodes (23): NewFarmHandler(), FarmCropRate, Farm, FarmListFilter, FarmListing, toModelFarmListing(), FarmService, NewFarmService() (+15 more)

### Community 12 - "auth_test.go"
Cohesion: 0.13
Nodes (29): net/http.Cookie, net/http.Handler, net/http/httptest.ResponseRecorder, ClaimsFromContext(), RequireAuth(), assertSameClaims(), assertUnauthorized(), newRequest() (+21 more)

### Community 13 - "Schwarzes Brett (Farmer Announcement Board)"
Cohesion: 0.13
Nodes (19): POST /api/announcements (Schwarzes Brett), services.Dispatcher, EmailSender (internal/repositories/email_sender.go), services.NotificationService, POST /api/notifications, Best-effort Async Delivery After Response, Board-not-Inbox Visibility Design Choice, Embedded Templates (embed.FS) Rationale (+11 more)

### Community 14 - "Three-Layer Architecture Pattern"
Cohesion: 0.12
Nodes (19): config.Load(), testcontainers integration testing, BaaS Backend Architecture, Containerfile Multi-stage Deployment, Known Gaps and Rough Edges (12 items), sqlc + dbmate Migrations-as-schema-of-record, Applying Three-Layer Pattern in Microservices, Three-Layer Architecture Pattern (+11 more)

### Community 15 - "Config"
Cohesion: 0.19
Nodes (19): Config, Load(), parseBoolEnv(), parseIntEnv(), TestParseIntEnv(), TestValidateSMTP_CredentialsMustBeSetInPairs(), TestValidateSMTP_DisabledSkipsTheCredentialRules(), TestValidateSMTP_EnabledRequiresHostAndSender() (+11 more)

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
Nodes (11): CompleteRentalCheckoutParams, InsertRentalCheckoutParams, github.com/google/uuid.UUID, Field, mapForeignKeyError(), mapGeometryError(), checkFieldOwnership(), fieldRepository (+3 more)

### Community 20 - "account.sql.go"
Cohesion: 0.16
Nodes (9): GetAccountByEmailRow, GetAccountByIDRow, GetAllRecipientsRow, InsertAccountParams, InsertAdminParams, InsertCustomerParams, InsertFarmerParams, ListAccountsParams (+1 more)

### Community 21 - "Service Layer (internal/services)"
Cohesion: 0.23
Nodes (12): Dropped ST_Equals Rectangle CHECK Constraint, SQLSTATE-to-HTTP Error Translation Table, mapGeometryError(), trg_plot_within_field Trigger (ST_Within), AccountHandler (example), AccountRepository (example), AccountService (example), Repository-interface Dependency Inversion (+4 more)

### Community 22 - "github.com/twpayne/go-geom.Polygon"
Cohesion: 0.11
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
Cohesion: 0.10
Nodes (5): Crop, toCrops(), cropRepository, fakeCropRepo, fakePaymentCropRepo

### Community 45 - "Rental"
Cohesion: 0.15
Nodes (8): time.Time, CheckoutSessionResult, Rental, RentalWithField, cropOffered(), fakePaymentRentalRepo, fakePaymentRentalService, fakeRentalRepo

### Community 46 - "main"
Cohesion: 0.07
Nodes (48): main(), migrateDB(), seedAdmin(), createRipenessNoticeRequest, createRipenessNoticeResponse, InboxHandler, inboxItemResponse, RipenessNoticeHandler (+40 more)

### Community 47 - "CareInstruction"
Cohesion: 0.08
Nodes (27): careInstructionRow, CareInstruction, CropAtFarm, CropCareGuide, PlotCareGuide, mapCareInstructionError(), toCareInstruction(), distinctGuides() (+19 more)

### Community 48 - "payment_service_test.go"
Cohesion: 0.08
Nodes (27): CheckoutSessionStatusResult, RentalCheckout, toModelRentalCheckout(), int32Ptr(), newTestPaymentService(), sessionIDFor(), TestCreateCheckoutSession_ComputesPriceAndRecordsAPendingCheckout(), TestCreateCheckoutSession_CropNotOfferedNeverReachesStripe() (+19 more)

### Community 49 - ".FindPostalCodeCoordinates"
Cohesion: 0.50
Nodes (3): PostalCode, github.com/twpayne/go-geom.Point, Queries

### Community 50 - "care_instruction.sql.go"
Cohesion: 0.11
Nodes (15): CopyDefaultCareInstructionsToFarmParams, DeleteFarmCareGuideParams, GetCareInstructionByIDRow, GetDefaultCareInstructionsByCropRow, GetEffectiveCareInstructionsParams, GetEffectiveCareInstructionsRow, GetFarmCopyOfCareInstructionParams, GetFarmCopyOfCareInstructionRow (+7 more)

### Community 53 - "announcement.sql.go"
Cohesion: 0.19
Nodes (8): GetAnnouncementsByFarmerRow, GetAnnouncementsForCustomerRow, GetCustomersOfFarmerForFieldRow, GetCustomersOfFarmerForPlotRow, GetCustomersOfFarmerRow, InsertAnnouncementParams, InsertAnnouncementRow, Queries

### Community 55 - "Role"
Cohesion: 0.27
Nodes (7): AccountListFilter, AccountListing, Role, Page, toModelAccountListing(), clampPagination(), T

### Community 57 - "farmer"
Cohesion: 0.21
Nodes (10): account, admin, customer, farmer, idx_account_email, idx_rental_customer, rental, announcement (+2 more)

### Community 58 - "fakePendingRegistrationRepo"
Cohesion: 0.21
Nodes (5): time.Duration, PendingRegistration, NewPendingRegistrationRepository(), pendingRegistrationRepository, fakePendingRegistrationRepo

### Community 59 - "Claims"
Cohesion: 0.22
Nodes (8): Claims, Issuer, NewIssuer(), TestIssueParseRoundTrip(), TestParseRejectsAlgNone(), TestParseRejectsForeignSecret(), TestParseRejectsWrongType(), jwt.RegisteredClaims

### Community 60 - "rental_checkout"
Cohesion: 0.26
Nodes (5): Queries, farm_crop_rate, idx_rental_checkout_customer, idx_rental_checkout_rental, rental_checkout

### Community 61 - "RentalStatus"
Cohesion: 0.23
Nodes (3): RentalStatus, mapRentalError(), rentalRepository

### Community 62 - "AuthService"
Cohesion: 0.29
Nodes (6): AuthService, RegisterInput, TokenPair, NewAuthService(), PendingRegistrationRepository, verifyEmailData

### Community 63 - "ActiveRental"
Cohesion: 0.20
Nodes (3): ActiveRental, RentalWithPlotAndCustomer, fakeCareRentalRepo

### Community 64 - "ripeness_notice.sql.go"
Cohesion: 0.29
Nodes (6): GetCustomersOfFarmerForFieldAndCropParams, GetCustomersOfFarmerForFieldAndCropRow, GetRipenessNoticesForCustomerRow, InsertRipenessNoticeParams, InsertRipenessNoticeRow, Queries

### Community 65 - "RipenessNoticeWithDetails"
Cohesion: 0.33
Nodes (5): RipenessNoticeWithDetails, toModelRipenessNotice(), RipenessNotice, ripenessNoticeRepository, fakeInboxRipenessNoticeRepo

### Community 66 - ".GetFarmCropRates"
Cohesion: 0.28
Nodes (4): GetFarmCropRateParams, GetFarmCropRatesRow, InsertFarmCropRateParams, Queries

### Community 67 - "account_handler.go"
Cohesion: 0.36
Nodes (7): AccountHandler, accountListingResponse, accountPageResponse, NewAccountHandler(), optionalRole(), toAccountPageResponse(), AccountService

### Community 68 - "PlotCropOffering"
Cohesion: 0.25
Nodes (4): PlotCropOffering, ComputeRentalPriceCents(), TestComputeRentalPriceCents(), TestComputeRentalPriceCents_MatchesTheDocumentedFormula()

### Community 69 - ".ListAccounts"
Cohesion: 0.50
Nodes (7): NewAccountService(), TestListAccounts_ClampsPagination(), TestListAccounts_OnlyAdminReachesTheRepository(), TestListAccounts_PassesThePageThrough(), TestListAccounts_RejectsUnknownRoleFilter(), TestListAccounts_TrimsTheSearchTerm(), TestListAccounts_WrapsRepositoryErrors()

### Community 70 - "newTestRentalService"
Cohesion: 0.46
Nodes (7): newTestRentalService(), TestRentalService_ApproveRental_ForbiddenWhenFarmerDoesNotOwnPlot(), TestRentalService_ApproveRental_OK(), TestRentalService_DeclineRental_AlreadyDecided(), TestRentalService_RequestRental_OK(), TestRentalService_RequestRental_RejectsBlankMessage(), TestRentalService_RequestRental_RejectsStartDateOutsideWindow()

### Community 71 - "care_instruction"
Cohesion: 0.43
Nodes (6): care_instruction, idx_care_instruction_crop_week, farm_care_guide, idx_care_instruction_crop_farm_week, idx_care_instruction_crop_week, idx_care_instruction_farm_based_on

### Community 72 - "HashPassword"
Cohesion: 0.43
Nodes (5): HashPassword(), TestHashUsesFreshSalt(), TestHashVerify(), TestVerifyRejectsMalformed(), VerifyPassword()

### Community 75 - "20260913114701_postal_codes.sql"
Cohesion: 0.70
Nodes (4): idx_postal_code_coordinates, idx_postal_code_name, idx_postal_code_zipcode, postal_code

### Community 76 - "20260915135248_drop_plot_check.sql"
Cohesion: 0.50
Nodes (3): check_plot_within_field(), check_plot_within_field, trg_plot_within_field

## Knowledge Gaps
- **30 isolated node(s):** `github.com/Neue-Konzepte-BaaS/backend`, `createAnnouncementRequest`, `loginRequest`, `registerRequest`, `verifyEmailRequest` (+25 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 90 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `main` to `net/http.ResponseWriter`, `NotificationService`, `AnnouncementWithFarm`, `account_handler.go`, `testing.T`, `Statistics`, `.ListAccounts`, `NewFarmService`, `Config`, `CareInstruction`, `email_sender_test.go`, `fakePendingRegistrationRepo`, `Claims`, `AuthService`?**
  _High betweenness centrality (0.050) - this node is a cross-community bridge._
- **Why does `crop` connect `crop` to `PlotCropOffering`, `care_instruction`, `Plot`, `RentalWithPlot`, `CareInstruction`, `rental_checkout`, `ActiveRental`?**
  _High betweenness centrality (0.050) - this node is a cross-community bridge._
- **Why does `MustClaimsFromContext()` connect `net/http.ResponseWriter` to `context.Context`, `Claims`, `auth_test.go`?**
  _High betweenness centrality (0.038) - this node is a cross-community bridge._
- **What connects `github.com/Neue-Konzepte-BaaS/backend`, `createAnnouncementRequest`, `loginRequest` to the rest of the system?**
  _30 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.05604719764011799 - nodes in this community are weakly interconnected._
- **Should `NotificationService` be split into smaller, more focused modules?**
  _Cohesion score 0.053613053613053616 - nodes in this community are weakly interconnected._
- **Should `AnnouncementWithFarm` be split into smaller, more focused modules?**
  _Cohesion score 0.13257575757575757 - nodes in this community are weakly interconnected._