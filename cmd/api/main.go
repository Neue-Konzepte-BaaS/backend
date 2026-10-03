package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/config"
	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/emailtemplates"
	"github.com/Neue-Konzepte-BaaS/backend/internal/handlers"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxgeom "github.com/twpayne/pgx-geom"
)

func migrateDB(dbUrl string) {
	u, urlErr := url.Parse(dbUrl)
	if urlErr != nil {
		panic("Could not parse database arguments to connect to db for migrations")
	}

	dbM := dbmate.New(u)
	dbM.MigrationsDir = []string{"./sql/migrations"}

	migrationErr := dbM.CreateAndMigrate()
	if migrationErr != nil {
		panic("Migration failed: " + migrationErr.Error())
	}
}

// notificationConcurrency caps how many notification fan-outs run at once.
// Each one sends serially, so this bounds the connections a burst of
// broadcasts can open against the relay.
const notificationConcurrency = 4

func seedAdmin(ctx context.Context, accountRepo services.AccountRepository, cfg config.Config) {
	if cfg.AdminEmail == "" {
		return
	}
	hash, err := credentials.HashPassword(cfg.AdminPassword)
	if err != nil {
		slog.Error("admin seed: hashing password failed", "error", err)
		return
	}
	_, err = accountRepo.CreateAdmin(ctx, models.Account{
		FirstName:    "Admin",
		LastName:     "BaaS",
		Email:        cfg.AdminEmail,
		PasswordHash: hash,
	})
	if errors.Is(err, services.ErrEmailTaken) {
		slog.Info("admin seed: account already exists, skipping", "email", cfg.AdminEmail)
		return
	}
	if err != nil {
		slog.Error("admin seed: failed", "error", err)
		return
	}
	slog.Info("admin seed: account created", "email", cfg.AdminEmail)
}

// seedSubscriptionPlanDefaults are placeholder prices and plot caps for the
// three tiers, seeded once at first boot. Correct them afterward via
// PUT /api/admin/subscription-plans/{planID}/price -- these numbers are
// deliberately not final, only enough to have real Stripe Price objects to
// point checkout sessions at from day one.
var seedSubscriptionPlanDefaults = []struct {
	code        models.SubscriptionPlanCode
	displayName string
	maxPlots    *int32
	priceCents  int32
}{
	{models.SubscriptionPlanCheap, "Cheap", int32Ptr(5), 990},
	{models.SubscriptionPlanModest, "Modest", int32Ptr(20), 2900},
	{models.SubscriptionPlanExpensive, "Expensive", nil, 9900},
}

func int32Ptr(v int32) *int32 { return &v }

// seedSubscriptionPlans creates the three fixed subscription tiers on first
// boot, same idempotent "skip if exists" shape as seedAdmin. Creating a
// Stripe Price requires calling Stripe, which a pure SQL migration cannot
// do -- that is why this seeding happens here instead of in the migration
// itself.
func seedSubscriptionPlans(ctx context.Context, planRepo services.SubscriptionPlanRepository, paymentGateway services.PaymentGateway) {
	for _, plan := range seedSubscriptionPlanDefaults {
		if _, err := planRepo.GetSubscriptionPlanByCode(ctx, plan.code); err == nil {
			continue
		} else if !errors.Is(err, services.ErrNotFound) {
			slog.Error("subscription plan seed: looking up plan failed", "code", plan.code, "error", err)
			continue
		}

		stripePriceID, err := paymentGateway.CreateSubscriptionPrice(ctx, plan.displayName, int64(plan.priceCents))
		if err != nil {
			slog.Error("subscription plan seed: creating stripe price failed", "code", plan.code, "error", err)
			continue
		}

		if err := planRepo.CreateSubscriptionPlan(ctx, plan.code, plan.displayName, plan.maxPlots, plan.priceCents, stripePriceID); err != nil {
			slog.Error("subscription plan seed: creating plan failed", "code", plan.code, "error", err)
			continue
		}
		slog.Info("subscription plan seed: plan created", "code", plan.code)
	}
}

// shutdownTimeout bounds both draining in-flight requests and waiting for
// background notification sends. smtp.SendMail has no timeout of its own, so
// without a deadline here a hung relay would keep the process alive.
//
// Note what this does and does not guarantee. A fan-out still in flight when
// the deadline passes is abandoned, and at one fresh SMTP connection per
// message a broadcast only finishes inside 15s for a small audience. Shutting
// down gracefully narrows the window in which queued mail is lost; it does not
// close it. Closing it needs delivery that survives the process — see the
// outbox note in ARCHITECTURE.md §9a.
const shutdownTimeout = 15 * time.Second

// newEmailSender picks the delivery backend. With SMTP disabled the console
// sender logs each message instead, so the whole notification path can be
// exercised locally and in tests without a relay.
func newEmailSender(c config.Config) services.EmailSender {
	if !c.SMTPEnabled {
		slog.Warn("SMTP is disabled; notifications will be logged instead of sent")
		return repositories.NewConsoleEmailSender()
	}

	return repositories.NewSMTPEmailSender(repositories.SMTPConfig{
		Hostname:    c.SMTPHost,
		Port:        c.SMTPPort,
		Username:    c.SMTPUsername,
		Password:    c.SMTPPassword,
		SenderName:  c.SMTPSenderName,
		SenderEmail: c.SMTPSenderEmail,
	})
}

func main() {
	// Structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	slog.Info("BaaS backend is starting...")

	// config
	c, configErr := config.Load()
	if configErr != nil {
		panic("Could not load config: " + configErr.Error())
	}

	// db migrations
	// Must run before the pool below: registering the PostGIS types needs the
	// postgis extension to already exist, which the field migration creates.
	if c.DBAutoMigrate {
		migrateDB(c.DatabaseURL)
	}

	// setup db connection pool
	ctx := context.Background()

	poolConfig, poolConfigErr := pgxpool.ParseConfig(c.DatabaseURL)
	if poolConfigErr != nil {
		panic("Could not parse database config: " + poolConfigErr.Error())
	}

	// register PostGIS geometry types on every new connection so that
	// geometry columns encode/decode as *geom.Polygon
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxgeom.Register(ctx, conn)
	}

	pool, poolErr := pgxpool.NewWithConfig(ctx, poolConfig)
	if poolErr != nil {
		panic("Could not connect to database: " + poolErr.Error())
	}
	defer pool.Close()

	if pingErr := pool.Ping(ctx); pingErr != nil {
		panic("Could not reach database: " + pingErr.Error())
	}

	// wiring: queries -> repositories -> services -> handlers -> routes
	queries := database.New(pool)

	accountRepo := repositories.NewAccountRepository(pool, queries)
	seedAdmin(ctx, accountRepo, c)
	farmRepo := repositories.NewFarmRepository(pool, queries)
	fieldRepo := repositories.NewFieldRepository(queries)
	plotRepo := repositories.NewPlotRepository(queries)
	postalCodeRepo := repositories.NewPostalCodeRepository(queries)
	rentalRepo := repositories.NewRentalRepository(queries)
	rentalCheckoutRepo := repositories.NewRentalCheckoutRepository(queries)
	announcementRepo := repositories.NewAnnouncementRepository(queries)
	careInstructionRepo := repositories.NewCareInstructionRepository(pool, queries)
	broadcastNotificationRepo := repositories.NewBroadcastNotificationRepository(queries)
	pendingRegistrationRepo := repositories.NewPendingRegistrationRepository(queries)
	cropRepo := repositories.NewCropRepository(pool, queries)
	seasonRepo := repositories.NewSeasonRepository(pool, queries)
	statisticsRepo := repositories.NewStatisticsRepository(queries)
	paymentGateway := repositories.NewStripeGateway(c.StripeSecretKey, c.StripeWebhookSecret)
	ripenessNoticeRepo := repositories.NewRipenessNoticeRepository(queries)
	subscriptionPlanRepo := repositories.NewSubscriptionPlanRepository(queries)
	farmerSubscriptionRepo := repositories.NewFarmerSubscriptionRepository(queries)
	seedSubscriptionPlans(ctx, subscriptionPlanRepo, paymentGateway)

	dispatcher := services.NewDispatcher(notificationConcurrency)

	accountService := services.NewAccountService(accountRepo, farmerSubscriptionRepo, paymentGateway)
	farmService := services.NewFarmService(farmRepo)
	notificationService := services.NewNotificationService(newEmailSender(c), accountRepo, broadcastNotificationRepo, emailtemplates.FS, dispatcher)
	authService := services.NewAuthService(accountRepo, pendingRegistrationRepo, credentials.NewIssuer(c.JWTSecret), notificationService, dispatcher, c.FrontendURL)
	announcementService := services.NewAnnouncementService(farmRepo, fieldRepo, plotRepo, announcementRepo, notificationService)
	careGuideService := services.NewCareGuideService(careInstructionRepo, rentalRepo, farmRepo)
	ripenessNoticeService := services.NewRipenessNoticeService(farmRepo, fieldRepo, ripenessNoticeRepo, notificationService)
	inboxService := services.NewInboxService(broadcastNotificationRepo, announcementRepo, ripenessNoticeRepo, careGuideService)
	subscriptionService := services.NewSubscriptionService(farmerSubscriptionRepo, subscriptionPlanRepo, paymentGateway, accountRepo, c.FrontendURL)
	fieldService := services.NewFieldService(farmRepo, fieldRepo, plotRepo, cropRepo)
	plotService := services.NewPlotService(farmRepo, fieldRepo, plotRepo, subscriptionService)
	plotSearchService := services.NewPlotSearchService(plotRepo, postalCodeRepo, cropRepo, seasonRepo)
	rentalService := services.NewRentalService(farmRepo, fieldRepo, rentalRepo, plotRepo, cropRepo, seasonRepo)
	cropService := services.NewCropService(farmRepo, fieldRepo, plotRepo, cropRepo)
	seasonService := services.NewSeasonService(seasonRepo, farmRepo, cropRepo)
	statisticsService := services.NewStatisticsService(farmRepo, statisticsRepo)
	paymentService := services.NewPaymentService(rentalService, rentalRepo, rentalCheckoutRepo, paymentGateway, plotRepo, cropRepo, fieldRepo, farmRepo, seasonRepo, c.FrontendURL)

	accountHandler := handlers.NewAccountHandler(accountService, c)
	authHandler := handlers.NewAuthHandler(authService, c)
	farmHandler := handlers.NewFarmHandler(farmService)
	fieldHandler := handlers.NewFieldHandler(fieldService, plotService)
	announcementHandler := handlers.NewAnnouncementHandler(announcementService)
	careGuideHandler := handlers.NewCareGuideHandler(careGuideService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	inboxHandler := handlers.NewInboxHandler(inboxService)
	plotSearchHandler := handlers.NewPlotSearchHandler(plotSearchService)
	rentalHandler := handlers.NewRentalHandler(rentalService, paymentService)
	cropHandler := handlers.NewCropHandler(cropService)
	seasonHandler := handlers.NewSeasonHandler(seasonService)
	statisticsHandler := handlers.NewStatisticsHandler(statisticsService)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionService)
	paymentHandler := handlers.NewPaymentHandler(paymentService, subscriptionService)
	ripenessNoticeHandler := handlers.NewRipenessNoticeHandler(ripenessNoticeService)

	router := handlers.NewRouter(accountHandler, authHandler, announcementHandler, careGuideHandler, farmHandler, fieldHandler, notificationHandler, inboxHandler, plotSearchHandler, rentalHandler, cropHandler, seasonHandler, statisticsHandler, paymentHandler, ripenessNoticeHandler, subscriptionHandler, authService, subscriptionService, c)

	// Shutdown is graceful because notifications are delivered after the
	// response is written: killing the process on SIGTERM would drop mail that
	// a caller has already been told is on its way.
	srv := &http.Server{Addr: ":8080", Handler: router}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "error", err)
			stop()
		}
	}()

	<-signalCtx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("draining requests failed", "error", err)
	}
	if err := dispatcher.Wait(shutdownCtx); err != nil {
		slog.Error("pending notifications were dropped", "error", err)
	}
}
