package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// checkFieldOwnership verifies that farmer's own farm is the one that owns
// field: resolve the caller's farm, resolve the field's farm, compare.
// Returns ErrForbidden on a mismatch, and passes ErrNotFound through
// unchanged when the field lookup can't find it.
//
// Shared by every service that scopes a write to a field:
// plotService.CreatePlot, announcementService.checkScopeOwnership (which
// resolves a plot to its field first), and
// ripenessNoticeService.CreateRipenessNotice. Before this was extracted, all
// three hand-implemented the same three steps, and had already begun to
// drift from one another.
func checkFieldOwnership(ctx context.Context, farmRepo FarmRepository, fieldRepo FieldRepository, farmer, field uuid.UUID) error {
	callerFarm, err := farmRepo.GetFarmIDByFarmerID(ctx, farmer)
	if err != nil {
		return fmt.Errorf("looking up farm: %w", err)
	}

	fieldFarm, err := fieldRepo.GetFieldFarm(ctx, field)
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
