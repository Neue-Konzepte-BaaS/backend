package repositories_test

import (
	"context"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
)

func TestBroadcastNotificationRepository_CreateAndListNewestFirst(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	repo := repositories.NewBroadcastNotificationRepository(queries)

	first, err := repo.CreateBroadcastNotification(ctx, "Wartung", "Sonntag offline")
	if err != nil {
		t.Fatalf("creating first broadcast: %v", err)
	}
	second, err := repo.CreateBroadcastNotification(ctx, "Update", "Neue Funktionen")
	if err != nil {
		t.Fatalf("creating second broadcast: %v", err)
	}

	all, err := repo.GetAllBroadcastNotifications(ctx)
	if err != nil {
		t.Fatalf("listing broadcasts: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("got %d broadcasts, want 2: %+v", len(all), all)
	}
	if all[0].ID != second.ID || all[1].ID != first.ID {
		t.Errorf("order = %+v, want the most recently created first", all)
	}
	if all[0].Subject != "Update" || all[0].Body != "Neue Funktionen" {
		t.Errorf("first broadcast = %+v, want the stored subject/body", all[0])
	}
}
