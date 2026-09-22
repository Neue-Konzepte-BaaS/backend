package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// AnnouncementService is the Schwarzes Brett: notices a farmer posts for the
// customers currently renting from him.
type AnnouncementService interface {
	// CreateAnnouncement stores the notice and then mails it to the audience
	// the scope selects: every current customer when field and plot are both
	// nil, or only the customers currently renting that field/plot when one
	// of them is set (the caller must not set both). The announcement is
	// persisted before any mail is queued, so a delivery failure still leaves
	// it readable on the board — which is the point of keeping a board at
	// all. The returned count is how many customers were queued, not how
	// many were reached.
	CreateAnnouncement(ctx context.Context, farmer uuid.UUID, subject, body string, field, plot *uuid.UUID) (models.AnnouncementWithFarm, int, error)
	// GetAnnouncementsForFarmer returns what this farmer has posted.
	GetAnnouncementsForFarmer(ctx context.Context, farmer uuid.UUID) ([]models.AnnouncementWithFarm, error)
	// GetAnnouncementsForCustomer returns the notices of every farmer the
	// customer currently rents from.
	GetAnnouncementsForCustomer(ctx context.Context, customer uuid.UUID) ([]models.AnnouncementWithFarm, error)
}

type announcementService struct {
	farmRepo            FarmRepository
	fieldRepo           FieldRepository
	plotRepo            PlotRepository
	announcementRepo    AnnouncementRepository
	notificationService NotificationService
}

// NewAnnouncementService wires the board to the notification provider.
//
// This is the first service in the codebase to depend on another service
// rather than only on repositories. Posting a notice is one business rule —
// store it, then tell the people it concerns — and ordering that in the
// handler would put business logic where the layering does not allow it.
// NotificationService is an interface owned by this package, so the dependency
// is still inverted; only the collaborator is a service rather than a table.
func NewAnnouncementService(farmRepo FarmRepository, fieldRepo FieldRepository, plotRepo PlotRepository, announcementRepo AnnouncementRepository, notificationService NotificationService) AnnouncementService {
	return &announcementService{
		farmRepo:            farmRepo,
		fieldRepo:           fieldRepo,
		plotRepo:            plotRepo,
		announcementRepo:    announcementRepo,
		notificationService: notificationService,
	}
}

func (s *announcementService) CreateAnnouncement(ctx context.Context, farmer uuid.UUID, subject, body string, field, plot *uuid.UUID) (models.AnnouncementWithFarm, int, error) {
	// The handler already rejects a request that sets both, but this
	// invariant belongs to the service, not the transport: checkScopeOwnership
	// below only ever validates field (silently skipping plot) when both are
	// non-nil, so a future caller that reaches this method directly — a test,
	// an admin tool, a reordered handler — must not be able to smuggle a
	// plot past its own ownership check.
	if field != nil && plot != nil {
		return models.AnnouncementWithFarm{}, 0, ErrInvalidFilter
	}

	if field != nil || plot != nil {
		if err := s.checkScopeOwnership(ctx, farmer, field, plot); err != nil {
			return models.AnnouncementWithFarm{}, 0, err
		}
	}

	announcement, err := s.announcementRepo.CreateAnnouncement(ctx, farmer, subject, body, field, plot)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.AnnouncementWithFarm{}, 0, err
		}
		return models.AnnouncementWithFarm{}, 0, fmt.Errorf("creating announcement: %w", err)
	}

	// The notice is already stored, so a failure here costs the mail, not the
	// announcement. Report it and return the announcement anyway rather than
	// telling the caller the post failed when it did not — the board still has
	// it, which is exactly the case a board exists for.
	queued, err := s.notificationService.NotifyFarmerCustomers(ctx, farmer, field, plot, announcement.FarmName, subject, body)
	if err != nil {
		slog.Error("announcement stored but could not be mailed",
			"announcement", announcement.ID, "farmer", farmer, "error", err)
		return announcement, 0, nil
	}

	return announcement, queued, nil
}

// checkScopeOwnership verifies that the farmer owns the field, or the field
// that the plot belongs to. It mirrors plotService.CreatePlot's ownership
// check: resolve the caller's own farm, resolve the scope's farm, and
// compare.
func (s *announcementService) checkScopeOwnership(ctx context.Context, farmer uuid.UUID, field, plot *uuid.UUID) error {
	callerFarm, err := s.farmRepo.GetFarmIDByFarmerID(ctx, farmer)
	if err != nil {
		return fmt.Errorf("looking up farm: %w", err)
	}

	scopedField := field
	if scopedField == nil {
		resolvedField, err := s.plotRepo.GetPlotField(ctx, *plot)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return err
			}
			return fmt.Errorf("looking up plot field: %w", err)
		}
		scopedField = &resolvedField
	}

	fieldFarm, err := s.fieldRepo.GetFieldFarm(ctx, *scopedField)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("looking up field farm: %w", err)
	}
	if fieldFarm != callerFarm {
		return ErrForbidden
	}
	return nil
}

func (s *announcementService) GetAnnouncementsForFarmer(ctx context.Context, farmer uuid.UUID) ([]models.AnnouncementWithFarm, error) {
	announcements, err := s.announcementRepo.GetAnnouncementsByFarmer(ctx, farmer)
	if err != nil {
		return nil, fmt.Errorf("getting announcements: %w", err)
	}
	return announcements, nil
}

func (s *announcementService) GetAnnouncementsForCustomer(ctx context.Context, customer uuid.UUID) ([]models.AnnouncementWithFarm, error) {
	announcements, err := s.announcementRepo.GetAnnouncementsForCustomer(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("getting announcements: %w", err)
	}
	return announcements, nil
}
