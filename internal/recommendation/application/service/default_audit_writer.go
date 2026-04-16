package service

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type DefaultAuditWriter struct {
	repository apprepo.RecommendationAuditRepository
}

var _ AuditWriter = (*DefaultAuditWriter)(nil)

func NewDefaultAuditWriter(repository apprepo.RecommendationAuditRepository) *DefaultAuditWriter {
	return &DefaultAuditWriter{repository: repository}
}

func (w *DefaultAuditWriter) Write(ctx context.Context, run model.RecommendationRun, items []model.RecommendationItem) error {
	if queries, ok := queriesFromContext(ctx); ok {
		if err := insertRunWithQueries(ctx, queries, run); err != nil {
			return err
		}
		for _, item := range items {
			if err := insertItemWithQueries(ctx, queries, item); err != nil {
				return err
			}
		}
		return nil
	}

	if err := w.repository.InsertRun(ctx, run); err != nil {
		return err
	}
	return w.repository.InsertItems(ctx, items)
}

func insertRunWithQueries(ctx context.Context, queries *recommendationsqlc.Queries, run model.RecommendationRun) error {
	pgRunID, err := mapper.StringToUUID(run.RunID)
	if err != nil {
		return err
	}
	pgUserID, err := mapper.StringToUUID(run.UserID)
	if err != nil {
		return err
	}

	return queries.InsertVideoRecommendationRun(ctx, recommendationsqlc.InsertVideoRecommendationRunParams{
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

func insertItemWithQueries(ctx context.Context, queries *recommendationsqlc.Queries, item model.RecommendationItem) error {
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

	return queries.InsertVideoRecommendationItem(ctx, recommendationsqlc.InsertVideoRecommendationItemParams{
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
