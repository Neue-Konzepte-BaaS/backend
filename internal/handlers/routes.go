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
func NewRouter(accountHandler *AccountHandler, authHandler *AuthHandler, announcementHandler *AnnouncementHandler, careGuideHandler *CareGuideHandler, farmHandler *FarmHandler, fieldHandler *FieldHandler, notificationHandler *NotificationHandler, inboxHandler *InboxHandler, plotSearchHandler *PlotSearchHandler, rentalHandler *RentalHandler, cropHandler *CropHandler, statisticsHandler *StatisticsHandler, ripenessNoticeHandler *RipenessNoticeHandler, authService services.AuthService, cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(middleware.Logger)

	// Browser clients live on a different origin (the frontend dev server), so
	// cross-origin requests must be allowed to carry the auth cookies.
	if cfg.CORSEnabled {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{cfg.FrontendURL},
			AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
			MaxAge:           300,
			// Response headers a cross-origin client may read beyond the
			// CORS-safelisted ones; see GetCareInstructions.
			ExposedHeaders: []string{careGuideSourceHeader},
		}))
	}

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/register", authHandler.Register)
		r.Post("/verify-email", authHandler.VerifyEmail)
		r.Post("/logout", authHandler.Logout)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authService))
			r.Get("/me", authHandler.Me)
		})
	})

	// Everything an admin reaches that is not an admin-only variant of an
	// existing route lives here, under one gate. The farm listing is here
	// rather than under /api/farms because it is a back-office view: it
	// carries the owner and the holdings, where GET /api/farms/{farmID} is
	// public and carries neither.
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireRole(models.RoleAdmin))

		r.Get("/accounts", accountHandler.ListAccounts)
		r.Get("/farms", farmHandler.ListFarms)
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

	// The tenant's own care guide: one entry per plot they are renting right
	// now, so the scope is the caller's rentals and needs no id in the path.
	r.Route("/api/care-guide", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireRole(models.RoleCustomer))

		r.Get("/", careGuideHandler.GetCareGuide)
	})

	// Editing one instruction is addressed by its own id rather than through
	// its crop: the crop is not needed to find it, and a path carrying both
	// would have to be checked for disagreeing about which crop it belongs to.
	// An admin edits the default guide, a farmer their farm's own version —
	// the service decides which guide a write lands in.
	r.Route("/api/care-instructions", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireAnyRole(models.RoleAdmin, models.RoleFarmer))

		r.Put("/{instructionID}", careGuideHandler.UpdateCareInstruction)
		r.Delete("/{instructionID}", careGuideHandler.DeleteCareInstruction)
	})

	r.Route("/api/farms", func(r chi.Router) {
		r.Get("/{farmID}", farmHandler.GetFarm)
	})

	r.Route("/api/fields", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireRole(models.RoleFarmer))

		r.Post("/", fieldHandler.CreateField)
		r.Get("/", fieldHandler.GetFields)
		r.Post("/{fieldID}/plots", fieldHandler.CreatePlot)
		r.Post("/{fieldID}/ripeness", ripenessNoticeHandler.Create)
	})

	r.Route("/api/notifications", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireRole(models.RoleAdmin))

		r.Post("/", notificationHandler.Broadcast)
	})

	r.Route("/api/inbox", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireRole(models.RoleCustomer))
		r.Get("/", inboxHandler.GetInbox)
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
			r.Delete("/{cropID}", cropHandler.DeleteCrop)
		})

		// A crop's whole guide, unscoped by any rental: the admin's view of
		// the default, and the farmer's view of the version their renters are
		// told. The catalog itself is public, but the advice is not — a
		// customer reads it through /api/care-guide, against their own week.
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authService))
			r.Use(appmiddleware.RequireAnyRole(models.RoleAdmin, models.RoleFarmer))
			r.Get("/{cropID}/care-instructions", careGuideHandler.GetCareInstructions)
			r.Post("/{cropID}/care-instructions", careGuideHandler.CreateCareInstruction)
		})

		// Resetting is a farmer's alone: the default guide has nothing to be
		// reset to.
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authService))
			r.Use(appmiddleware.RequireRole(models.RoleFarmer))
			r.Delete("/{cropID}/farm-care-guide", careGuideHandler.ResetFarmCareGuide)
		})
	})

	r.Route("/api/rentals", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireRole(models.RoleCustomer))
			r.Post("/", rentalHandler.RentPlot)
			r.Get("/", rentalHandler.GetRentals)
		})

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireRole(models.RoleFarmer))
			r.Get("/farm", rentalHandler.GetFarmRentals)
			r.Post("/{rentalID}/approve", rentalHandler.ApproveRental)
			r.Post("/{rentalID}/decline", rentalHandler.DeclineRental)
		})
	})

	r.Route("/api/statistics", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(authService))
		r.Use(appmiddleware.RequireAnyRole(models.RoleFarmer, models.RoleAdmin))

		r.Get("/", statisticsHandler.GetStatistics)
	})

	return r
}
