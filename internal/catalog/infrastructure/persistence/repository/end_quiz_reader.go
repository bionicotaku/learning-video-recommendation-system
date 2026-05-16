package repository

import (
	"context"
	"errors"
	"fmt"

	"learning-video-recommendation-system/internal/catalog/domain/model"
	"learning-video-recommendation-system/internal/catalog/infrastructure/persistence/mapper"
	catalogsqlc "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EndQuizQuestionReader struct {
	pool *pgxpool.Pool
}

func NewEndQuizQuestionReader(pool *pgxpool.Pool) *EndQuizQuestionReader {
	return &EndQuizQuestionReader{pool: pool}
}

func (r *EndQuizQuestionReader) HasVisibleVideoForEndQuiz(ctx context.Context, videoID string) (bool, error) {
	if r.pool == nil {
		return false, errors.New("pg pool is required")
	}
	id, err := mapper.StringToUUID(videoID)
	if err != nil {
		return false, fmt.Errorf("map video_id: %w", err)
	}
	return catalogsqlc.New(r.pool).HasVisibleVideoForEndQuiz(ctx, id)
}

func (r *EndQuizQuestionReader) ListVideoUnitQuizQuestionCandidates(ctx context.Context, videoID string, coarseUnitIDs []int64) ([]model.EndQuizQuestionCandidate, error) {
	if r.pool == nil {
		return nil, errors.New("pg pool is required")
	}
	if len(coarseUnitIDs) == 0 {
		return nil, nil
	}
	id, err := mapper.StringToUUID(videoID)
	if err != nil {
		return nil, fmt.Errorf("map video_id: %w", err)
	}
	rows, err := catalogsqlc.New(r.pool).ListVideoUnitQuizQuestionCandidates(ctx, catalogsqlc.ListVideoUnitQuizQuestionCandidatesParams{
		VideoID:       id,
		CoarseUnitIds: coarseUnitIDs,
	})
	if err != nil {
		return nil, err
	}
	result := make([]model.EndQuizQuestionCandidate, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.EndQuizQuestionCandidate{
			QuestionID:           mapper.UUIDToString(row.QuestionID),
			ScopeType:            row.ScopeType,
			QuestionType:         row.QuestionType,
			CoarseUnitID:         row.CoarseUnitID,
			TargetText:           row.TargetText,
			ContextSentenceIndex: int32Pointer(row.ContextSentenceIndex),
			ContextSpanIndex:     int32Pointer(row.ContextSpanIndex),
			ContextStartMS:       int32Pointer(row.ContextStartMs),
			ContextEndMS:         int32Pointer(row.ContextEndMs),
			ContentPayload:       row.ContentPayload,
		})
	}
	return result, nil
}

func (r *EndQuizQuestionReader) ListUnitQuizQuestionCandidates(ctx context.Context, coarseUnitIDs []int64) ([]model.EndQuizQuestionCandidate, error) {
	if r.pool == nil {
		return nil, errors.New("pg pool is required")
	}
	if len(coarseUnitIDs) == 0 {
		return nil, nil
	}
	rows, err := catalogsqlc.New(r.pool).ListUnitQuizQuestionCandidates(ctx, coarseUnitIDs)
	if err != nil {
		return nil, err
	}
	result := make([]model.EndQuizQuestionCandidate, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.EndQuizQuestionCandidate{
			QuestionID:           mapper.UUIDToString(row.QuestionID),
			ScopeType:            row.ScopeType,
			QuestionType:         row.QuestionType,
			CoarseUnitID:         row.CoarseUnitID,
			TargetText:           row.TargetText,
			ContextSentenceIndex: int32Pointer(row.ContextSentenceIndex),
			ContextSpanIndex:     int32Pointer(row.ContextSpanIndex),
			ContextStartMS:       int32Pointer(row.ContextStartMs),
			ContextEndMS:         int32Pointer(row.ContextEndMs),
			ContentPayload:       row.ContentPayload,
		})
	}
	return result, nil
}

func int32Pointer(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	result := value.Int32
	return &result
}
