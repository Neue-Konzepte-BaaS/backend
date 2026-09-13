package handlers

import (
	"net/http"

	appmiddleware "github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter chains up all routes located in the different handlers
func NewRouter(authHandler *AuthHandler, fieldHandler *FieldHandler, authService services.AuthService) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authService))
			r.Get("/me", authHandler.Me)
		})
	})

	r.Route("/api/fields", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireRole(models.RoleFarmer))

		r.Post("/", fieldHandler.CreateField)
		r.Get("/", fieldHandler.GetFields)
		r.Post("/{fieldID}/plots", fieldHandler.CreatePlot)
	})

	return r
}
