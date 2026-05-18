package service

import (
	"context"
	"errors"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

type SetVideoFavoriteUsecase struct {
	writer apprepo.VideoInteractionWriter
}

func NewSetVideoFavoriteUsecase(writer apprepo.VideoInteractionWriter) *SetVideoFavoriteUsecase {
	return &SetVideoFavoriteUsecase{writer: writer}
}

func (u *SetVideoFavoriteUsecase) Execute(ctx context.Context, request dto.SetVideoFavoriteRequest) (dto.VideoFavoriteResponse, error) {
	if u.writer == nil {
		return dto.VideoFavoriteResponse{}, errors.New("video interaction writer is required")
	}
	if request.UserID == "" {
		return dto.VideoFavoriteResponse{}, validationError("user_id is required")
	}
	if request.VideoID == "" {
		return dto.VideoFavoriteResponse{}, validationError("video_id is required")
	}

	result, err := u.writer.SetVideoFavorite(ctx, model.VideoFavoriteCommand{
		UserID:  request.UserID,
		VideoID: request.VideoID,
		Enabled: request.Enabled,
	})
	if err != nil {
		if errors.Is(err, apprepo.ErrVideoNotFound) {
			return dto.VideoFavoriteResponse{}, NotFoundError("video not found")
		}
		return dto.VideoFavoriteResponse{}, err
	}

	return dto.VideoFavoriteResponse{
		VideoID:       result.VideoID,
		HasFavorited:  result.HasFavorited,
		FavoriteCount: result.FavoriteCount,
	}, nil
}
