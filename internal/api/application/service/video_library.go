package service

import (
	"context"
	"fmt"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogusecase "learning-video-recommendation-system/internal/catalog/application/usecase"
)

type VideoLibraryService struct {
	favorites  catalogusecase.ListVideoFavoritesUsecase
	history    catalogusecase.ListVideoHistoryUsecase
	urlBuilder PublicAssetURLBuilder
}

func NewVideoLibraryService(
	favorites catalogusecase.ListVideoFavoritesUsecase,
	history catalogusecase.ListVideoHistoryUsecase,
	urlBuilder PublicAssetURLBuilder,
) *VideoLibraryService {
	return &VideoLibraryService{favorites: favorites, history: history, urlBuilder: urlBuilder}
}

func (s *VideoLibraryService) ListFavorites(ctx context.Context, request apvdto.ListVideoFavoritesRequest) (apvdto.ListVideoFavoritesResponse, error) {
	if request.UserID == "" {
		return apvdto.ListVideoFavoritesResponse{}, InvalidRequestError("user_id is required")
	}
	if s.favorites == nil {
		return apvdto.ListVideoFavoritesResponse{}, fmt.Errorf("video favorites list usecase is required")
	}
	result, err := s.favorites.Execute(ctx, catalogdto.ListVideoFavoritesRequest{
		UserID: request.UserID,
		Limit:  request.Limit,
		Cursor: request.Cursor,
	})
	if err != nil {
		return apvdto.ListVideoFavoritesResponse{}, err
	}
	items := make([]apvdto.VideoFavoriteItem, 0, len(result.Items))
	for _, item := range result.Items {
		coverURL, err := optionalPublicAssetURL(s.urlBuilder, item.CoverImageURL)
		if err != nil {
			return apvdto.ListVideoFavoritesResponse{}, fmt.Errorf("build cover_image_url for video favorite: video_id=%s: %w", item.VideoID, err)
		}
		items = append(items, apvdto.VideoFavoriteItem{
			VideoID:         item.VideoID,
			Title:           item.Title,
			CoverImageURL:   coverURL,
			DurationSeconds: durationSeconds(item.DurationMS),
			ViewCount:       item.ViewCount,
			FavoritedAt:     item.FavoritedAt,
		})
	}
	return apvdto.ListVideoFavoritesResponse{
		Items: items,
		Page: apvdto.VideoLibraryPage{
			Limit:      result.Page.Limit,
			HasMore:    result.Page.HasMore,
			NextCursor: result.Page.NextCursor,
		},
	}, nil
}

func (s *VideoLibraryService) ListHistory(ctx context.Context, request apvdto.ListVideoHistoryRequest) (apvdto.ListVideoHistoryResponse, error) {
	if request.UserID == "" {
		return apvdto.ListVideoHistoryResponse{}, InvalidRequestError("user_id is required")
	}
	if s.history == nil {
		return apvdto.ListVideoHistoryResponse{}, fmt.Errorf("video history list usecase is required")
	}
	result, err := s.history.Execute(ctx, catalogdto.ListVideoHistoryRequest{
		UserID: request.UserID,
		Limit:  request.Limit,
		Cursor: request.Cursor,
	})
	if err != nil {
		return apvdto.ListVideoHistoryResponse{}, err
	}
	items := make([]apvdto.VideoHistoryItem, 0, len(result.Items))
	for _, item := range result.Items {
		coverURL, err := optionalPublicAssetURL(s.urlBuilder, item.CoverImageURL)
		if err != nil {
			return apvdto.ListVideoHistoryResponse{}, fmt.Errorf("build cover_image_url for video history: video_id=%s: %w", item.VideoID, err)
		}
		items = append(items, apvdto.VideoHistoryItem{
			VideoID:         item.VideoID,
			Title:           item.Title,
			CoverImageURL:   coverURL,
			DurationSeconds: durationSeconds(item.DurationMS),
			ViewCount:       item.ViewCount,
			LastPositionMS:  item.LastPositionMS,
			LastWatchedAt:   item.LastWatchedAt,
		})
	}
	return apvdto.ListVideoHistoryResponse{
		Items: items,
		Page: apvdto.VideoLibraryPage{
			Limit:      result.Page.Limit,
			HasMore:    result.Page.HasMore,
			NextCursor: result.Page.NextCursor,
		},
	}, nil
}
