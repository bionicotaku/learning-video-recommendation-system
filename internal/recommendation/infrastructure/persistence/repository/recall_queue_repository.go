package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type RecallQueueRepository struct {
	db      recommendationsqlc.DBTX
	queries *recommendationsqlc.Queries
}

var _ apprepo.RecallQueueRepository = (*RecallQueueRepository)(nil)

func NewRecallQueueRepository(db recommendationsqlc.DBTX) *RecallQueueRepository {
	return &RecallQueueRepository{
		db:      db,
		queries: recommendationsqlc.New(db),
	}
}

func (r *RecallQueueRepository) GetLearningStateVersion(ctx context.Context, userID string) (apprepo.LearningStateVersion, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return apprepo.LearningStateVersion{}, err
	}
	row, err := r.queries.GetLearningStateVersionForRecommendation(ctx, pgUserID)
	if err != nil {
		return apprepo.LearningStateVersion{}, err
	}
	return apprepo.LearningStateVersion{
		ActiveTargetUnitCount:      row.ActiveTargetUnitCount,
		SourceLearningMaxUpdatedAt: mapper.TimePointerFromPG(row.SourceLearningMaxUpdatedAt),
	}, nil
}

func (r *RecallQueueRepository) GetProjectionUpdatedAt(ctx context.Context) (time.Time, error) {
	value, err := r.queries.GetRecallProjectionMetadata(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return mapper.TimeFromPG(value), nil
}

func (r *RecallQueueRepository) GetQueueState(ctx context.Context, userID string) (model.RecallQueueState, bool, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return model.RecallQueueState{}, false, err
	}
	row, err := r.queries.GetUserRecallQueueState(ctx, pgUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.RecallQueueState{}, false, nil
		}
		return model.RecallQueueState{}, false, err
	}
	return toRecallQueueState(row), true, nil
}

func (r *RecallQueueRepository) RebuildUserQueue(ctx context.Context, userID string, projectionUpdatedAt time.Time) (model.RecallQueueState, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return model.RecallQueueState{}, err
	}

	if pool, ok := r.db.(*pgxpool.Pool); ok {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return model.RecallQueueState{}, err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback(ctx)
			}
		}()

		state, err := rebuildUserQueueLocked(ctx, tx, userID, pgUserID, projectionUpdatedAt)
		if err != nil {
			return model.RecallQueueState{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return model.RecallQueueState{}, err
		}
		committed = true
		return state, nil
	}

	return rebuildUserQueueLocked(ctx, r.db, userID, pgUserID, projectionUpdatedAt)
}

func rebuildUserQueueLocked(ctx context.Context, db recommendationsqlc.DBTX, userID string, pgUserID pgtype.UUID, projectionUpdatedAt time.Time) (model.RecallQueueState, error) {
	if _, err := db.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1, 0))`, userID); err != nil {
		return model.RecallQueueState{}, err
	}
	row, err := recommendationsqlc.New(db).RebuildUserUnitRecallQueue(ctx, recommendationsqlc.RebuildUserUnitRecallQueueParams{
		UserID:                    pgUserID,
		SourceProjectionUpdatedAt: mapper.TimePointerToPG(&projectionUpdatedAt),
	})
	if err != nil {
		return model.RecallQueueState{}, err
	}
	return toRecallQueueState(row), nil
}

func (r *RecallQueueRepository) ListCandidates(ctx context.Context, userID string, now time.Time, suppliedPerBucketLimit int32, noSupplyPerBucketLimit int32) ([]model.RecallQueueCandidate, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListUserRecallQueueCandidates(ctx, recommendationsqlc.ListUserRecallQueueCandidatesParams{
		UserID:                 pgUserID,
		NowAt:                  mapper.TimePointerToPG(&now),
		SuppliedPerBucketLimit: suppliedPerBucketLimit,
		NoSupplyPerBucketLimit: noSupplyPerBucketLimit,
	})
	if err != nil {
		return nil, err
	}
	result := make([]model.RecallQueueCandidate, 0, len(rows))
	for _, row := range rows {
		mapped, err := toRecallQueueCandidate(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func toRecallQueueState(row recommendationsqlc.RecommendationUserUnitRecallQueueState) model.RecallQueueState {
	return model.RecallQueueState{
		UserID:                     mapper.UUIDToString(row.UserID),
		SourceLearningMaxUpdatedAt: mapper.TimePointerFromPG(row.SourceLearningMaxUpdatedAt),
		SourceProjectionUpdatedAt:  mapper.TimeFromPG(row.SourceProjectionUpdatedAt),
		ActiveTargetUnitCount:      row.ActiveTargetUnitCount,
		RebuiltAt:                  mapper.TimeFromPG(row.RebuiltAt),
	}
}

func toRecallQueueCandidate(row recommendationsqlc.ListUserRecallQueueCandidatesRow) (model.RecallQueueCandidate, error) {
	targetPriority, err := mapper.NumericToFloat64(row.TargetPriority)
	if err != nil {
		return model.RecallQueueCandidate{}, err
	}
	masteryScore, err := mapper.NumericToFloat64(row.MasteryScore)
	if err != nil {
		return model.RecallQueueCandidate{}, err
	}
	dynamicPriority, err := mapper.NumericToFloat64(row.DynamicPriority)
	if err != nil {
		return model.RecallQueueCandidate{}, err
	}
	return model.RecallQueueCandidate{
		UserID:              mapper.UUIDToString(row.UserID),
		CoarseUnitID:        row.CoarseUnitID,
		Status:              row.Status,
		TargetPriority:      targetPriority,
		MasteryScore:        masteryScore,
		LastProgressQuality: mapper.Int16PointerFromPG(row.LastProgressQuality),
		NextReviewAt:        mapper.TimePointerFromPG(row.NextReviewAt),
		SupplyGrade:         row.SupplyGrade,
		StateUpdatedAt:      mapper.TimeFromPG(row.StateUpdatedAt),
		LastServedAt:        mapper.TimePointerFromPG(row.LastServedAt),
		ServedCount:         row.ServedCount,
		Bucket:              row.Bucket,
		DynamicPriority:     dynamicPriority,
		BucketRank:          row.BucketRank,
	}, nil
}
