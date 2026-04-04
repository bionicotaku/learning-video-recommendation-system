package integration

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	infra "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure"
	repopkg "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestUserUnitStateRepositoryCandidateQueries(t *testing.T) {
	cfg := infra.LoadConfig()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := infra.NewDBPool(ctx, cfg)
	if err != nil {
		t.Fatalf("NewDBPool() error = %v", err)
	}
	defer pool.Close()

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := sqlcgen.New(tx)
	repo := repopkg.NewUserUnitStateRepository()

	userID, err := loadExistingUserID(ctx, tx)
	if err != nil {
		t.Fatalf("loadExistingUserID() error = %v", err)
	}

	unitIDs, err := loadAvailableCoarseUnitIDs(ctx, tx, userID, 3)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDs() error = %v", err)
	}

	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)

	insertStates := []*model.UserUnitState{
		{
			UserID:          userID,
			CoarseUnitID:    unitIDs[0],
			IsTarget:        true,
			Status:          enum.UnitStatusReviewing,
			TargetPriority:  0.9,
			ProgressPercent: 40,
			MasteryScore:    0.4,
			EaseFactor:      2.5,
			Repetition:      2,
			IntervalDays:    3,
			NextReviewAt:    ptrTime(now.Add(-1 * time.Hour)),
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			UserID:          userID,
			CoarseUnitID:    unitIDs[1],
			IsTarget:        true,
			Status:          enum.UnitStatusNew,
			TargetPriority:  0.8,
			ProgressPercent: 0,
			MasteryScore:    0,
			EaseFactor:      2.5,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			UserID:          userID,
			CoarseUnitID:    unitIDs[2],
			IsTarget:        true,
			Status:          enum.UnitStatusLearning,
			TargetPriority:  0.7,
			ProgressPercent: 10,
			MasteryScore:    0.1,
			EaseFactor:      2.5,
			Repetition:      1,
			IntervalDays:    1,
			NextReviewAt:    ptrTime(now.Add(2 * time.Hour)),
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}

	if err := repo.BatchUpsert(ctx, q, insertStates); err != nil {
		t.Fatalf("BatchUpsert() error = %v", err)
	}

	reviewCandidates, err := repo.FindDueReviewCandidates(ctx, q, userID, now)
	if err != nil {
		t.Fatalf("FindDueReviewCandidates() error = %v", err)
	}
	if len(reviewCandidates) != 1 {
		t.Fatalf("len(reviewCandidates) = %d, want 1", len(reviewCandidates))
	}
	if reviewCandidates[0].State.CoarseUnitID != unitIDs[0] {
		t.Fatalf("review candidate coarseUnitID = %d, want %d", reviewCandidates[0].State.CoarseUnitID, unitIDs[0])
	}
	if reviewCandidates[0].Unit.Kind != enum.UnitKindWord && reviewCandidates[0].Unit.Kind != enum.UnitKindPhrase && reviewCandidates[0].Unit.Kind != enum.UnitKindGrammar {
		t.Fatalf("review candidate kind = %q, want supported kind", reviewCandidates[0].Unit.Kind)
	}

	newCandidates, err := repo.FindNewCandidates(ctx, q, userID)
	if err != nil {
		t.Fatalf("FindNewCandidates() error = %v", err)
	}
	if len(newCandidates) != 1 {
		t.Fatalf("len(newCandidates) = %d, want 1", len(newCandidates))
	}
	if newCandidates[0].State.CoarseUnitID != unitIDs[1] {
		t.Fatalf("new candidate coarseUnitID = %d, want %d", newCandidates[0].State.CoarseUnitID, unitIDs[1])
	}
	if newCandidates[0].Unit.Kind != enum.UnitKindWord && newCandidates[0].Unit.Kind != enum.UnitKindPhrase && newCandidates[0].Unit.Kind != enum.UnitKindGrammar {
		t.Fatalf("new candidate kind = %q, want supported kind", newCandidates[0].Unit.Kind)
	}
}

func loadExistingUserID(ctx context.Context, tx pgx.Tx) (uuid.UUID, error) {
	var userID uuid.UUID
	err := tx.QueryRow(ctx, `select id from auth.users limit 1`).Scan(&userID)
	return userID, err
}

func loadAvailableCoarseUnitIDs(ctx context.Context, tx pgx.Tx, userID uuid.UUID, count int) ([]int64, error) {
	rows, err := tx.Query(ctx, `
		select c.id
		from semantic.coarse_unit c
		left join learning.user_unit_states s
		  on s.coarse_unit_id = c.id
		 and s.user_id = $1
		where s.coarse_unit_id is null
		order by c.id
		limit $2
	`, userID, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) < count {
		return nil, pgx.ErrNoRows
	}

	return ids, nil
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
