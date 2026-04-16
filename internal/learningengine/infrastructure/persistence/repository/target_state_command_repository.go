package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/mapper"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
)

type TargetStateCommandRepository struct {
	queries *learningenginesqlc.Queries
}

var _ apprepo.TargetStateCommandRepository = (*TargetStateCommandRepository)(nil)

func NewTargetStateCommandRepository(db learningenginesqlc.DBTX) *TargetStateCommandRepository {
	return &TargetStateCommandRepository{
		queries: learningenginesqlc.New(db),
	}
}

func (r *TargetStateCommandRepository) EnsureTargetUnits(ctx context.Context, userID string, targets []model.TargetUnitSpec) error {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return err
	}

	for _, target := range targets {
		targetPriority, err := mapper.Float64ToNumeric(target.TargetPriority)
		if err != nil {
			return err
		}

		if err := r.queries.EnsureTargetUnit(ctx, learningenginesqlc.EnsureTargetUnitParams{
			UserID:            pgUserID,
			CoarseUnitID:      target.CoarseUnitID,
			TargetSource:      mapper.StringToText(target.TargetSource),
			TargetSourceRefID: mapper.StringToText(target.TargetSourceRefID),
			TargetPriority:    targetPriority,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *TargetStateCommandRepository) SetTargetInactive(ctx context.Context, userID string, coarseUnitID int64) error {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return err
	}

	return r.queries.SetTargetInactive(ctx, learningenginesqlc.SetTargetInactiveParams{
		UserID:       pgUserID,
		CoarseUnitID: coarseUnitID,
	})
}

func (r *TargetStateCommandRepository) SuspendTargetUnit(ctx context.Context, userID string, coarseUnitID int64, reason string) error {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return err
	}

	return r.queries.SuspendTargetUnit(ctx, learningenginesqlc.SuspendTargetUnitParams{
		UserID:          pgUserID,
		CoarseUnitID:    coarseUnitID,
		SuspendedReason: mapper.StringToText(reason),
	})
}

func (r *TargetStateCommandRepository) ResumeTargetUnit(ctx context.Context, userID string, coarseUnitID int64) error {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return err
	}

	return r.queries.ResumeTargetUnit(ctx, learningenginesqlc.ResumeTargetUnitParams{
		UserID:       pgUserID,
		CoarseUnitID: coarseUnitID,
	})
}
