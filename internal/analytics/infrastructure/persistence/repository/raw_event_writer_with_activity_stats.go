package repository

import (
	"context"
	"errors"

	apprepo "learning-video-recommendation-system/internal/analytics/application/repository"
	"learning-video-recommendation-system/internal/analytics/domain/model"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	userpersist "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RawEventWriterWithActivityStats struct {
	pool *pgxpool.Pool
}

var _ apprepo.RawEventWriter = (*RawEventWriterWithActivityStats)(nil)

func NewRawEventWriterWithActivityStats(pool *pgxpool.Pool) *RawEventWriterWithActivityStats {
	return &RawEventWriterWithActivityStats{pool: pool}
}

func (w *RawEventWriterWithActivityStats) UpsertLearningInteractions(ctx context.Context, events []model.RawLearningInteractionEvent) ([]model.RawEventWriteResult, error) {
	if w.pool == nil {
		return nil, errors.New("pg pool is required")
	}
	return withinAnalyticsStatsTx(ctx, w.pool, func(tx pgx.Tx, stats userrepo.ActivityStatsRecorder) ([]model.RawEventWriteResult, error) {
		results, err := NewRawEventWriter(tx).UpsertLearningInteractions(ctx, events)
		if err != nil {
			return nil, err
		}
		for index, result := range results {
			if !result.Inserted {
				continue
			}
			event := events[index]
			if event.EventType != "exposure" && event.EventType != "lookup" {
				continue
			}
			if err := stats.IncrementLearningInteraction(ctx, event.UserID, event.OccurredAt); err != nil {
				return nil, err
			}
		}
		return results, nil
	})
}

func (w *RawEventWriterWithActivityStats) UpsertQuizEvent(ctx context.Context, event model.RawQuizEvent) (model.RawEventWriteResult, error) {
	if w.pool == nil {
		return model.RawEventWriteResult{}, errors.New("pg pool is required")
	}
	return withinAnalyticsStatsTx(ctx, w.pool, func(tx pgx.Tx, stats userrepo.ActivityStatsRecorder) (model.RawEventWriteResult, error) {
		result, err := NewRawEventWriter(tx).UpsertQuizEvent(ctx, event)
		if err != nil {
			return model.RawEventWriteResult{}, err
		}
		if result.Inserted {
			if err := stats.IncrementQuizAttempt(ctx, event.UserID, event.CompletedAt); err != nil {
				return model.RawEventWriteResult{}, err
			}
			if err := stats.IncrementLearningInteraction(ctx, event.UserID, event.CompletedAt); err != nil {
				return model.RawEventWriteResult{}, err
			}
		}
		return result, nil
	})
}

func withinAnalyticsStatsTx[T any](ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx, stats userrepo.ActivityStatsRecorder) (T, error)) (T, error) {
	var zero T
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return zero, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	result, err := fn(tx, userpersist.NewRepository(tx))
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, err
	}
	return result, nil
}
