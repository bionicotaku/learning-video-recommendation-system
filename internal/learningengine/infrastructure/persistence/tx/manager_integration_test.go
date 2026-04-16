package tx_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/application/service"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	persisttx "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/tx"
	"learning-video-recommendation-system/internal/learningengine/testutil"
)

func TestManagerRollsBackTransactionOnError(t *testing.T) {
	db := testutil.StartPostgres(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	manager := persisttx.NewManager(db.Pool)
	expectedErr := errors.New("force rollback")
	q4 := int16(4)

	err := manager.WithinTx(context.Background(), func(ctx context.Context, repos service.TransactionalRepositories) error {
		return repos.UnitLearningEvents().Append(ctx, []model.LearningEvent{
			{
				UserID:       userID,
				CoarseUnitID: 101,
				EventType:    "new_learn",
				SourceType:   "quiz_session",
				Quality:      &q4,
				Metadata:     []byte("{}"),
				OccurredAt:   time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
			},
		})
	})
	if err != nil {
		t.Fatalf("WithinTx() error on append = %v", err)
	}

	err = manager.WithinTx(context.Background(), func(ctx context.Context, repos service.TransactionalRepositories) error {
		if err := repos.UnitLearningEvents().Append(ctx, []model.LearningEvent{
			{
				UserID:       userID,
				CoarseUnitID: 101,
				EventType:    "review",
				SourceType:   "quiz_session",
				Quality:      &q4,
				Metadata:     []byte("{}"),
				OccurredAt:   time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC),
			},
		}); err != nil {
			return err
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("WithinTx() error = %v, want expectedErr", err)
	}

	var count int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.unit_learning_events where user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("count events error = %v", err)
	}
	if count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}
}
