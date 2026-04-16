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

func (r *RecommendationAuditRepository) InsertItem(ctx context.Context, item model.RecommendationItem) error {
	pgRunID, err := mapper.StringToUUID(item.RunID)
	if err != nil {
		return err
	}
	pgVideoID, err := mapper.StringToUUID(item.VideoID)
	if err != nil {
		return err
	}
	score, err := mapper.Float64ToNumeric(item.Score)
	if err != nil {
		return err
	}

	return r.queries.InsertVideoRecommendationItem(ctx, recommendationsqlc.InsertVideoRecommendationItemParams{
		RunID:                     pgRunID,
		Rank:                      item.Rank,
		VideoID:                   pgVideoID,
		Score:                     score,
		PrimaryLane:               mapper.StringToText(item.PrimaryLane),
		DominantBucket:            mapper.StringToText(item.DominantBucket),
		DominantUnitID:            mapper.Int64PointerToPG(item.DominantUnitID),
		ReasonCodes:               item.ReasonCodes,
		CoveredHardReviewCount:    item.CoveredHardReviewCount,
		CoveredNewNowCount:        item.CoveredNewNowCount,
		CoveredSoftReviewCount:    item.CoveredSoftReviewCount,
		CoveredNearFutureCount:    item.CoveredNearFutureCount,
		BestEvidenceSentenceIndex: mapper.Int32PointerToPG(item.BestEvidenceSentenceIndex),
		BestEvidenceSpanIndex:     mapper.Int32PointerToPG(item.BestEvidenceSpanIndex),
		BestEvidenceStartMs:       mapper.Int32PointerToPG(item.BestEvidenceStartMs),
		BestEvidenceEndMs:         mapper.Int32PointerToPG(item.BestEvidenceEndMs),
	})
}
