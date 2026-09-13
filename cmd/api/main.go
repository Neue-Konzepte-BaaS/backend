package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/Neue-Konzepte-BaaS/backend/internal/config"
	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
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
	cropRepo := repositories.NewCropRepository(pool, queries)
	statisticsRepo := repositories.NewStatisticsRepository(queries)

	authService := services.NewAuthService(accountRepo, credentials.NewIssuer(c.JWTSecret))
	fieldService := services.NewFieldService(fieldRepo, plotRepo, cropRepo)
	plotService := services.NewPlotService(fieldRepo, plotRepo)
	plotSearchService := services.NewPlotSearchService(plotRepo, postalCodeRepo)
	rentalService := services.NewRentalService(rentalRepo, plotRepo, cropRepo)
	cropService := services.NewCropService(fieldRepo, cropRepo)
	statisticsService := services.NewStatisticsService(statisticsRepo)

	authHandler := handlers.NewAuthHandler(authService, c)
	fieldHandler := handlers.NewFieldHandler(fieldService, plotService)
	plotSearchHandler := handlers.NewPlotSearchHandler(plotSearchService)
	rentalHandler := handlers.NewRentalHandler(rentalService)
	cropHandler := handlers.NewCropHandler(cropService)
	statisticsHandler := handlers.NewStatisticsHandler(statisticsService)

	router := handlers.NewRouter(authHandler, fieldHandler, plotSearchHandler, rentalHandler, cropHandler, statisticsHandler, authService, c)

	slog.Info("listening", "addr", ":8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
