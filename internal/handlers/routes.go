package handlers

import (
	"net/http"

	"github.com/Neue-Konzepte-BaaS/backend/internal/config"
	appmiddleware "github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter chains up all routes located in the different handlers
func NewRouter(authHandler *AuthHandler, authService services.AuthService, cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(middleware.Logger)

	// Browser clients live on a different origin (the frontend dev server), so
	// cross-origin requests must be allowed to carry the auth cookies.
	if cfg.CORSEnabled {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{cfg.FrontendURL},
			AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/register", authHandler.Register)
		r.Post("/logout", authHandler.Logout)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authService))
			r.Get("/me", authHandler.Me)
		})
	})

	return r
}
