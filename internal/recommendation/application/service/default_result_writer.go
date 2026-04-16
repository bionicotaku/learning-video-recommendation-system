package service

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/tx"
)

type DefaultRecommendationResultWriter struct {
	txManager     *tx.Manager
	auditWriter   AuditWriter
	servingStates ServingStateManager
}

var _ RecommendationResultWriter = (*DefaultRecommendationResultWriter)(nil)

func NewDefaultRecommendationResultWriter(
	txManager *tx.Manager,
	auditWriter AuditWriter,
	servingStates ServingStateManager,
) *DefaultRecommendationResultWriter {
	return &DefaultRecommendationResultWriter{
		txManager:     txManager,
		auditWriter:   auditWriter,
		servingStates: servingStates,
	}
}

func (w *DefaultRecommendationResultWriter) Persist(ctx context.Context, run model.RecommendationRun, items []model.RecommendationItem, userID string, videos []model.FinalRecommendationItem) error {
	return w.txManager.WithinTx(ctx, func(txCtx context.Context, queries *recommendationsqlc.Queries) error {
		txCtx = WithQueries(txCtx, queries)
		if err := w.auditWriter.Write(txCtx, run, items); err != nil {
			return err
		}
		return w.servingStates.ApplySelection(txCtx, run.RunID, userID, videos)
	})
}
