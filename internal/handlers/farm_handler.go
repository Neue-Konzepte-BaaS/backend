package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type FarmHandler struct {
	farmService services.FarmService
}

func NewFarmHandler(farmService services.FarmService) *FarmHandler {
	return &FarmHandler{farmService: farmService}
}

type farmResponse struct {
	ID                string  `json:"id"`
	FarmerID          string  `json:"farmerId"`
	Name              string  `json:"name"`
	Address           string  `json:"address"`
	Description       string  `json:"description"`
	FoundedAt         *string `json:"foundedAt"`
	TotalSquareMeters float64 `json:"totalSquareMeters"`
}

// GetFarm returns public details of the farm with the given id, including
// the total area of plots it offers on the platform. It is a public endpoint
// (no auth required) — customers browsing plots use it.
func (h *FarmHandler) GetFarm(w http.ResponseWriter, r *http.Request) {
	farmID, err := uuid.Parse(chi.URLParam(r, "farmID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid farm id")
		return
	}

	farm, err := h.farmService.GetFarm(r.Context(), farmID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "farm not found")
		return
	}
	if err != nil {
		slog.Error("getting farm failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var foundedAt *string
	if farm.FoundedAt != nil {
		s := farm.FoundedAt.Format(time.DateOnly)
		foundedAt = &s
	}

	webutils.WriteJSON(w, http.StatusOK, farmResponse{
		ID:                farm.ID.String(),
		FarmerID:          farm.FarmerID.String(),
		Name:              farm.Name,
		Address:           farm.Address,
		Description:       farm.Description,
		FoundedAt:         foundedAt,
		TotalSquareMeters: farm.TotalSquareMeters,
	})
}
