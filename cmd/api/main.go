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
	"github.com/jackc/pgx/v5/pgxpool"
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
	migrateDB(c.DatabaseURL)

	// setup db connection pool
	ctx := context.Background()

	pool, poolErr := pgxpool.New(ctx, c.DatabaseURL)
	if poolErr != nil {
		panic("Could not connect to database: " + poolErr.Error())
	}
	defer pool.Close()

	if pingErr := pool.Ping(ctx); pingErr != nil {
		panic("Could not reach database: " + pingErr.Error())
	}

	// wiring: queries -> repositories -> services -> handlers -> routes
	queries := database.New(pool)

	accountRepo := repositories.NewAccountRepository(queries)

	authService := services.NewAuthService(accountRepo, credentials.NewIssuer(c.JWTSecret))

	authHandler := handlers.NewAuthHandler(authService, c)

	router := handlers.NewRouter(authHandler, authService)

	slog.Info("listening", "addr", ":8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
