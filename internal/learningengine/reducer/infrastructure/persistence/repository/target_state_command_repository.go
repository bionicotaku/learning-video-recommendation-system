package repository

import (
	"context"
	"encoding/json"

	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/mapper"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/sqlcgen"
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
	if len(targets) == 0 {
		return nil
	}

	payload := make([]targetUnitSpecJSON, 0, len(targets))
	for _, target := range targets {
		payload = append(payload, targetUnitSpecJSON{
			CoarseUnitID:      target.CoarseUnitID,
			TargetSource:      target.TargetSource,
			TargetSourceRefID: target.TargetSourceRefID,
			TargetPriority:    target.TargetPriority,
		})
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return r.queries.EnsureTargetUnits(ctx, learningenginesqlc.EnsureTargetUnitsParams{
		UserID:  pgUserID,
		Targets: rawPayload,
	})
}

type targetUnitSpecJSON struct {
	CoarseUnitID      int64   `json:"coarse_unit_id"`
	TargetSource      string  `json:"target_source"`
	TargetSourceRefID string  `json:"target_source_ref_id"`
	TargetPriority    float64 `json:"target_priority"`
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
