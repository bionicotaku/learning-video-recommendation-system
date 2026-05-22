package service

import (
	"context"
	"fmt"
	"strings"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	appusecase "learning-video-recommendation-system/internal/catalog/application/usecase"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

const (
	defaultVideoLibraryLimit = 20
	maxVideoLibraryLimit     = 100
)

type ListVideoFavoritesUsecase struct {
	reader apprepo.VideoLibraryReader
}

var _ appusecase.ListVideoFavoritesUsecase = (*ListVideoFavoritesUsecase)(nil)

func NewListVideoFavoritesUsecase(reader apprepo.VideoLibraryReader) *ListVideoFavoritesUsecase {
	return &ListVideoFavoritesUsecase{reader: reader}
}

func (u *ListVideoFavoritesUsecase) Execute(ctx context.Context, request dto.ListVideoFavoritesRequest) (dto.ListVideoFavoritesResponse, error) {
	userID := strings.TrimSpace(request.UserID)
	if userID == "" {
		return dto.ListVideoFavoritesResponse{}, validationError("user_id is required")
	}
	if !isUUID(userID) {
		return dto.ListVideoFavoritesResponse{}, validationError("user_id must be a uuid")
	}
	if u.reader == nil {
		return dto.ListVideoFavoritesResponse{}, fmt.Errorf("video library reader is required")
	}
	limit, err := normalizeVideoLibraryLimit(request.Limit)
	if err != nil {
		return dto.ListVideoFavoritesResponse{}, err
	}
	cursor, err := decodeVideoLibraryCursor(request.Cursor, dto.VideoLibraryCursorKindFavorites)
	if err != nil {
		return dto.ListVideoFavoritesResponse{}, err
	}

	rows, err := u.reader.ListVideoFavorites(ctx, dto.ListVideoFavoritesQuery{
		UserID:       userID,
		LimitPlusOne: limit + 1,
		Cursor:       cursor,
	})
	if err != nil {
		return dto.ListVideoFavoritesResponse{}, err
	}

	hasMore := len(rows) > limit
	items := rows
	if hasMore {
		items = rows[:limit]
	}
	if items == nil {
		items = []model.VideoFavoriteListItem{}
	}

	var nextCursor *string
	if hasMore && len(items) > 0 {
		encoded, err := encodeVideoLibraryCursor(dto.VideoLibraryCursorKindFavorites, items[len(items)-1].FavoritedAt, items[len(items)-1].VideoID)
		if err != nil {
			return dto.ListVideoFavoritesResponse{}, fmt.Errorf("encode video favorites cursor: %w", err)
		}
		nextCursor = &encoded
	}

	return dto.ListVideoFavoritesResponse{
		Items: mapVideoFavoriteListItems(items),
		Page: dto.VideoLibraryPage{
			Limit:      limit,
			HasMore:    hasMore,
			NextCursor: nextCursor,
		},
	}, nil
}

type ListVideoHistoryUsecase struct {
	reader apprepo.VideoLibraryReader
}

var _ appusecase.ListVideoHistoryUsecase = (*ListVideoHistoryUsecase)(nil)

func NewListVideoHistoryUsecase(reader apprepo.VideoLibraryReader) *ListVideoHistoryUsecase {
	return &ListVideoHistoryUsecase{reader: reader}
}

func (u *ListVideoHistoryUsecase) Execute(ctx context.Context, request dto.ListVideoHistoryRequest) (dto.ListVideoHistoryResponse, error) {
	userID := strings.TrimSpace(request.UserID)
	if userID == "" {
		return dto.ListVideoHistoryResponse{}, validationError("user_id is required")
	}
	if !isUUID(userID) {
		return dto.ListVideoHistoryResponse{}, validationError("user_id must be a uuid")
	}
	if u.reader == nil {
		return dto.ListVideoHistoryResponse{}, fmt.Errorf("video library reader is required")
	}
	limit, err := normalizeVideoLibraryLimit(request.Limit)
	if err != nil {
		return dto.ListVideoHistoryResponse{}, err
	}
	cursor, err := decodeVideoLibraryCursor(request.Cursor, dto.VideoLibraryCursorKindHistory)
	if err != nil {
		return dto.ListVideoHistoryResponse{}, err
	}

	rows, err := u.reader.ListVideoHistory(ctx, dto.ListVideoHistoryQuery{
		UserID:       userID,
		LimitPlusOne: limit + 1,
		Cursor:       cursor,
	})
	if err != nil {
		return dto.ListVideoHistoryResponse{}, err
	}

	hasMore := len(rows) > limit
	items := rows
	if hasMore {
		items = rows[:limit]
	}
	if items == nil {
		items = []model.VideoHistoryListItem{}
	}

	var nextCursor *string
	if hasMore && len(items) > 0 {
		encoded, err := encodeVideoLibraryCursor(dto.VideoLibraryCursorKindHistory, items[len(items)-1].LastWatchedAt, items[len(items)-1].VideoID)
		if err != nil {
			return dto.ListVideoHistoryResponse{}, fmt.Errorf("encode video history cursor: %w", err)
		}
		nextCursor = &encoded
	}

	return dto.ListVideoHistoryResponse{
		Items: mapVideoHistoryListItems(items),
		Page: dto.VideoLibraryPage{
			Limit:      limit,
			HasMore:    hasMore,
			NextCursor: nextCursor,
		},
	}, nil
}

func normalizeVideoLibraryLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultVideoLibraryLimit, nil
	}
	if limit < 1 || limit > maxVideoLibraryLimit {
		return 0, validationError("limit must be between 1 and 100")
	}
	return limit, nil
}

func mapVideoFavoriteListItems(items []model.VideoFavoriteListItem) []dto.VideoFavoriteItem {
	result := make([]dto.VideoFavoriteItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.VideoFavoriteItem{
			VideoID:       item.VideoID,
			Title:         item.Title,
			CoverImageURL: item.CoverImageURL,
			DurationMS:    item.DurationMS,
			ViewCount:     item.ViewCount,
			FavoritedAt:   item.FavoritedAt,
		})
	}
	return result
}

func mapVideoHistoryListItems(items []model.VideoHistoryListItem) []dto.VideoHistoryItem {
	result := make([]dto.VideoHistoryItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.VideoHistoryItem{
			VideoID:        item.VideoID,
			Title:          item.Title,
			CoverImageURL:  item.CoverImageURL,
			DurationMS:     item.DurationMS,
			ViewCount:      item.ViewCount,
			LastPositionMS: item.LastPositionMS,
			LastWatchedAt:  item.LastWatchedAt,
		})
	}
	return result
}
