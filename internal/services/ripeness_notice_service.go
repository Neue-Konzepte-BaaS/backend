package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// RipenessNoticeService lets a farmer tell the customers growing a given crop
// on one of his plots that it is ready to harvest.
type RipenessNoticeService interface {
	// CreateRipenessNotice checks that the plot's field belongs to the
	// farmer, then stores the notice and mails everyone currently renting
	// that plot with that crop. The returned count is how many were queued,
	// not how many were reached — mirrors AnnouncementService.CreateAnnouncement.
	CreateRipenessNotice(ctx context.Context, farmer, plot, crop uuid.UUID) (models.RipenessNoticeWithDetails, int, error)
}

type ripenessNoticeService struct {
	farmRepo            FarmRepository
	fieldRepo           FieldRepository
	plotRepo            PlotRepository
	ripenessNoticeRepo  RipenessNoticeRepository
	notificationService NotificationService
}

func NewRipenessNoticeService(farmRepo FarmRepository, fieldRepo FieldRepository, plotRepo PlotRepository, ripenessNoticeRepo RipenessNoticeRepository, notificationService NotificationService) RipenessNoticeService {
	return &ripenessNoticeService{
		farmRepo:            farmRepo,
		fieldRepo:           fieldRepo,
		plotRepo:            plotRepo,
		ripenessNoticeRepo:  ripenessNoticeRepo,
		notificationService: notificationService,
	}
}

func (s *ripenessNoticeService) CreateRipenessNotice(ctx context.Context, farmer, plot, crop uuid.UUID) (models.RipenessNoticeWithDetails, int, error) {
	field, err := s.plotRepo.GetPlotField(ctx, plot)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.RipenessNoticeWithDetails{}, 0, err
		}
		return models.RipenessNoticeWithDetails{}, 0, fmt.Errorf("looking up plot field: %w", err)
	}
	if err := checkFieldOwnership(ctx, s.farmRepo, s.fieldRepo, farmer, field); err != nil {
		return models.RipenessNoticeWithDetails{}, 0, err
	}

	notice, err := s.ripenessNoticeRepo.CreateRipenessNotice(ctx, farmer, plot, crop)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.RipenessNoticeWithDetails{}, 0, err
		}
		return models.RipenessNoticeWithDetails{}, 0, fmt.Errorf("creating ripeness notice: %w", err)
	}

	// The notice is already stored, so a failure here costs the mail, not the
	// notice — same trade AnnouncementService makes, for the same reason.
	queued, err := s.notificationService.NotifyRipeness(ctx, plot, crop, notice.FarmName, notice.PlotName, notice.CropName)
	if err != nil {
		slog.Error("ripeness notice stored but could not be mailed",
			"notice", notice.ID, "plot", plot, "crop", crop, "error", err)
		return notice, 0, nil
	}

	return notice, queued, nil
}
