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

type FeedLookupReader struct {
	pool *pgxpool.Pool
}

func NewFeedLookupReader(pool *pgxpool.Pool) *FeedLookupReader {
	return &FeedLookupReader{pool: pool}
}

func (r *FeedLookupReader) ListFeedVideosByIDs(ctx context.Context, videoIDs []string) ([]model.FeedVideoDisplay, error) {
	if r.pool == nil {
		return nil, errors.New("pg pool is required")
	}
	if len(videoIDs) == 0 {
		return nil, nil
	}

	ids := make([]pgtype.UUID, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		id, err := mapper.StringToUUID(videoID)
		if err != nil {
			return nil, fmt.Errorf("map video_id: %w", err)
		}
		ids = append(ids, id)
	}

	rows, err := catalogsqlc.New(r.pool).ListFeedVideosByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]model.FeedVideoDisplay, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.FeedVideoDisplay{
			VideoID:               mapper.UUIDToString(row.VideoID),
			Title:                 row.Title,
			Description:           row.Description,
			HLSMasterPlaylistPath: row.HlsMasterPlaylistPath,
			CoverImageURL:         textPointer(row.ThumbnailUrl),
			ViewCount:             row.ViewCount,
			LikeCount:             row.LikeCount,
			FavoriteCount:         row.FavoriteCount,
		})
	}
	return result, nil
}

func (r *FeedLookupReader) ListUnitLabelsByIDs(ctx context.Context, coarseUnitIDs []int64) ([]model.UnitLabel, error) {
	if r.pool == nil {
		return nil, errors.New("pg pool is required")
	}
	if len(coarseUnitIDs) == 0 {
		return nil, nil
	}

	rows, err := catalogsqlc.New(r.pool).ListUnitLabelsByIDs(ctx, coarseUnitIDs)
	if err != nil {
		return nil, err
	}
	result := make([]model.UnitLabel, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.UnitLabel{
			CoarseUnitID: row.ID,
			Text:         row.Label,
		})
	}
	return result, nil
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}
