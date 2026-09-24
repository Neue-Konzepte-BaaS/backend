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
// on one of his fields that it is ready to harvest.
type RipenessNoticeService interface {
	// CreateRipenessNotice checks that the field belongs to the farmer, then
	// stores the notice and mails everyone currently renting a plot of that
	// field with that crop. The returned count is how many were queued, not
	// how many were reached — mirrors AnnouncementService.CreateAnnouncement.
	CreateRipenessNotice(ctx context.Context, farmer, field, crop uuid.UUID) (models.RipenessNoticeWithDetails, int, error)
}

type ripenessNoticeService struct {
	farmRepo            FarmRepository
	fieldRepo           FieldRepository
	ripenessNoticeRepo  RipenessNoticeRepository
	notificationService NotificationService
}

func NewRipenessNoticeService(farmRepo FarmRepository, fieldRepo FieldRepository, ripenessNoticeRepo RipenessNoticeRepository, notificationService NotificationService) RipenessNoticeService {
	return &ripenessNoticeService{
		farmRepo:            farmRepo,
		fieldRepo:           fieldRepo,
		ripenessNoticeRepo:  ripenessNoticeRepo,
		notificationService: notificationService,
	}
}

func (s *ripenessNoticeService) CreateRipenessNotice(ctx context.Context, farmer, field, crop uuid.UUID) (models.RipenessNoticeWithDetails, int, error) {
	if err := checkFieldOwnership(ctx, s.farmRepo, s.fieldRepo, farmer, field); err != nil {
		return models.RipenessNoticeWithDetails{}, 0, err
	}

	notice, err := s.ripenessNoticeRepo.CreateRipenessNotice(ctx, farmer, field, crop)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.RipenessNoticeWithDetails{}, 0, err
		}
		return models.RipenessNoticeWithDetails{}, 0, fmt.Errorf("creating ripeness notice: %w", err)
	}

	// The notice is already stored, so a failure here costs the mail, not the
	// notice — same trade AnnouncementService makes, for the same reason.
	queued, err := s.notificationService.NotifyRipeness(ctx, field, crop, notice.FarmName, notice.FieldName, notice.CropName)
	if err != nil {
		slog.Error("ripeness notice stored but could not be mailed",
			"notice", notice.ID, "field", field, "crop", crop, "error", err)
		return notice, 0, nil
	}

	return notice, queued, nil
}
