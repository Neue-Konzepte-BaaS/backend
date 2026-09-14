package handlers

import (
	"net/http"

	"github.com/Neue-Konzepte-BaaS/backend/internal/config"
	appmiddleware "github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter chains up all routes located in the different handlers
func NewRouter(authHandler *AuthHandler, announcementHandler *AnnouncementHandler, fieldHandler *FieldHandler, notificationHandler *NotificationHandler, plotSearchHandler *PlotSearchHandler, rentalHandler *RentalHandler, cropHandler *CropHandler, statisticsHandler *StatisticsHandler, authService services.AuthService, cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(middleware.Logger)

	// Browser clients live on a different origin (the frontend dev server), so
	// cross-origin requests must be allowed to carry the auth cookies.
	if cfg.CORSEnabled {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{cfg.FrontendURL},
			AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodOptions},
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

	r.Route("/api/announcements", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireRole(models.RoleFarmer))
			r.Post("/", announcementHandler.Create)
		})

		// Both sides read the same board, from opposite ends: a farmer sees
		// what he posted, a customer what the farmers he rents from posted.
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAnyRole(models.RoleFarmer, models.RoleCustomer))
			r.Get("/", announcementHandler.List)
		})
	})

	r.Route("/api/fields", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireRole(models.RoleFarmer))

		r.Post("/", fieldHandler.CreateField)
		r.Get("/", fieldHandler.GetFields)
		r.Post("/{fieldID}/plots", fieldHandler.CreatePlot)
	})

	r.Route("/api/notifications", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireRole(models.RoleAdmin))

		r.Post("/", notificationHandler.Broadcast)
	})

	r.Route("/api/plots", func(r chi.Router) {
		r.Get("/nearest", plotSearchHandler.FindNearestPlots)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authService))
			r.Use(appmiddleware.RequireRole(models.RoleFarmer))
			r.Put("/{plotID}/crops", cropHandler.SetPlotCrops)
		})
	})

	r.Route("/api/crops", func(r chi.Router) {
		r.Get("/", cropHandler.GetAllCrops)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authService))
			r.Use(appmiddleware.RequireRole(models.RoleAdmin))
			r.Post("/", cropHandler.CreateCrop)
		})
	})

	r.Route("/api/rentals", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireRole(models.RoleCustomer))

		r.Post("/", rentalHandler.RentPlot)
		r.Get("/", rentalHandler.GetRentals)
	})

	r.Route("/api/statistics", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireAnyRole(models.RoleFarmer, models.RoleAdmin))

		r.Get("/", statisticsHandler.GetStatistics)
	})

	return r
}
