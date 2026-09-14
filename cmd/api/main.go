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
	migrateDB(c.DatabaseURL)

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
	fieldRepo := repositories.NewFieldRepository(queries)
	plotRepo := repositories.NewPlotRepository(queries)
	postalCodeRepo := repositories.NewPostalCodeRepository(queries)
	rentalRepo := repositories.NewRentalRepository(queries)
	announcementRepo := repositories.NewAnnouncementRepository(queries)
	cropRepo := repositories.NewCropRepository(pool, queries)
	statisticsRepo := repositories.NewStatisticsRepository(queries)

	dispatcher := services.NewDispatcher(notificationConcurrency)

	authService := services.NewAuthService(accountRepo, credentials.NewIssuer(c.JWTSecret))
	fieldService := services.NewFieldService(fieldRepo, plotRepo, cropRepo)
	notificationService := services.NewNotificationService(newEmailSender(c), accountRepo, emailtemplates.FS, dispatcher)
	announcementService := services.NewAnnouncementService(announcementRepo, notificationService)
	plotService := services.NewPlotService(fieldRepo, plotRepo)
	plotSearchService := services.NewPlotSearchService(plotRepo, postalCodeRepo)
	rentalService := services.NewRentalService(rentalRepo, plotRepo, cropRepo)
	cropService := services.NewCropService(fieldRepo, cropRepo)
	statisticsService := services.NewStatisticsService(statisticsRepo)

	authHandler := handlers.NewAuthHandler(authService, c)
	fieldHandler := handlers.NewFieldHandler(fieldService, plotService)
	announcementHandler := handlers.NewAnnouncementHandler(announcementService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	plotSearchHandler := handlers.NewPlotSearchHandler(plotSearchService)
	rentalHandler := handlers.NewRentalHandler(rentalService)
	cropHandler := handlers.NewCropHandler(cropService)
	statisticsHandler := handlers.NewStatisticsHandler(statisticsService)

	router := handlers.NewRouter(authHandler, announcementHandler, fieldHandler, notificationHandler, plotSearchHandler, rentalHandler, cropHandler, statisticsHandler, authService, c)

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
