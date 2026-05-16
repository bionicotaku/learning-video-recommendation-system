package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type RecommendationAuditRepository struct {
	queries *recommendationsqlc.Queries
}

var _ apprepo.RecommendationAuditRepository = (*RecommendationAuditRepository)(nil)

func NewRecommendationAuditRepository(db recommendationsqlc.DBTX) *RecommendationAuditRepository {
	return &RecommendationAuditRepository{queries: recommendationsqlc.New(db)}
}

func (r *RecommendationAuditRepository) InsertRun(ctx context.Context, run model.RecommendationRun) error {
	pgRunID, err := mapper.StringToUUID(run.RunID)
	if err != nil {
		return err
	}
	pgUserID, err := mapper.StringToUUID(run.UserID)
	if err != nil {
		return err
	}

	return r.queries.InsertVideoRecommendationRun(ctx, recommendationsqlc.InsertVideoRecommendationRunParams{
		RunID:              pgRunID,
		UserID:             pgUserID,
		RequestContext:     run.RequestContext,
		SessionMode:        mapper.StringToText(run.SessionMode),
		SelectorMode:       mapper.StringToText(run.SelectorMode),
		PlannerSnapshot:    run.PlannerSnapshot,
		LaneBudgetSnapshot: run.LaneBudgetSnapshot,
		CandidateSummary:   run.CandidateSummary,
		Underfilled:        run.Underfilled,
		ResultCount:        run.ResultCount,
	})
}

func (r *RecommendationAuditRepository) InsertItems(ctx context.Context, items []model.RecommendationItem) error {
	if len(items) == 0 {
		return nil
	}
	payload, err := mapper.RecommendationItemsToJSON(items)
	if err != nil {
		return err
	}
	return r.queries.InsertVideoRecommendationItems(ctx, payload)
}
