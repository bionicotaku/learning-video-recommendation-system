package repository

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
	"learning-video-recommendation-system/internal/catalog/infrastructure/persistence/mapper"
	catalogsqlc "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

type VideoLibraryReader struct {
	queries *catalogsqlc.Queries
}

var _ apprepo.VideoLibraryReader = (*VideoLibraryReader)(nil)

func NewVideoLibraryReader(db catalogsqlc.DBTX) *VideoLibraryReader {
	return &VideoLibraryReader{queries: catalogsqlc.New(db)}
}

func (r *VideoLibraryReader) ListVideoFavorites(ctx context.Context, query dto.ListVideoFavoritesQuery) ([]model.VideoFavoriteListItem, error) {
	userID, err := mapper.StringToUUID(query.UserID)
	if err != nil {
		return nil, fmt.Errorf("map user_id: %w", err)
	}
	cursorVideoID, err := cursorVideoID(query.Cursor)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListVideoFavorites(ctx, catalogsqlc.ListVideoFavoritesParams{
		UserID:        userID,
		HasCursor:     query.Cursor != nil,
		CursorAt:      cursorAt(query.Cursor),
		CursorVideoID: cursorVideoID,
		LimitPlusOne:  int32(query.LimitPlusOne),
	})
	if err != nil {
		return nil, err
	}
	result := make([]model.VideoFavoriteListItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.VideoFavoriteListItem{
			VideoID:       mapper.UUIDToString(row.VideoID),
			Title:         row.Title,
			CoverImageURL: textPointer(row.ThumbnailUrl),
			DurationMS:    row.DurationMs,
			ViewCount:     row.ViewCount,
			FavoritedAt:   mapper.TimeFromPG(row.BookmarkedAt),
		})
	}
	return result, nil
}

func (r *VideoLibraryReader) ListVideoHistory(ctx context.Context, query dto.ListVideoHistoryQuery) ([]model.VideoHistoryListItem, error) {
	userID, err := mapper.StringToUUID(query.UserID)
	if err != nil {
		return nil, fmt.Errorf("map user_id: %w", err)
	}
	cursorVideoID, err := cursorVideoID(query.Cursor)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListVideoHistory(ctx, catalogsqlc.ListVideoHistoryParams{
		UserID:        userID,
		HasCursor:     query.Cursor != nil,
		CursorAt:      cursorAt(query.Cursor),
		CursorVideoID: cursorVideoID,
		LimitPlusOne:  int32(query.LimitPlusOne),
	})
	if err != nil {
		return nil, err
	}
	result := make([]model.VideoHistoryListItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.VideoHistoryListItem{
			VideoID:        mapper.UUIDToString(row.VideoID),
			Title:          row.Title,
			CoverImageURL:  textPointer(row.ThumbnailUrl),
			DurationMS:     row.DurationMs,
			ViewCount:      row.ViewCount,
			LastPositionMS: row.LastPositionMs,
			LastWatchedAt:  mapper.TimeFromPG(row.LastWatchedAt),
		})
	}
	return result, nil
}

func cursorAt(cursor *dto.VideoLibraryCursor) pgtype.Timestamptz {
	if cursor == nil {
		return pgtype.Timestamptz{}
	}
	return mapper.TimePointerToPG(&cursor.SortAt)
}

func cursorVideoID(cursor *dto.VideoLibraryCursor) (pgtype.UUID, error) {
	if cursor == nil {
		return pgtype.UUID{}, nil
	}
	value, err := mapper.StringToUUID(cursor.VideoID)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("map cursor video_id: %w", err)
	}
	return value, nil
}
