package service

import (
	"context"
	"errors"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

type SetVideoLikeUsecase struct {
	writer apprepo.VideoInteractionWriter
}

func NewSetVideoLikeUsecase(writer apprepo.VideoInteractionWriter) *SetVideoLikeUsecase {
	return &SetVideoLikeUsecase{writer: writer}
}

func (u *SetVideoLikeUsecase) Execute(ctx context.Context, request dto.SetVideoLikeRequest) (dto.VideoLikeResponse, error) {
	if u.writer == nil {
		return dto.VideoLikeResponse{}, errors.New("video interaction writer is required")
	}
	if request.UserID == "" {
		return dto.VideoLikeResponse{}, validationError("user_id is required")
	}
	if request.VideoID == "" {
		return dto.VideoLikeResponse{}, validationError("video_id is required")
	}
	if request.OccurredAt.IsZero() {
		return dto.VideoLikeResponse{}, validationError("occurred_at is required")
	}

	result, err := u.writer.SetVideoLike(ctx, model.VideoLikeCommand{
		UserID:     request.UserID,
		VideoID:    request.VideoID,
		Enabled:    request.Enabled,
		OccurredAt: request.OccurredAt.UTC(),
	})
	if err != nil {
		if errors.Is(err, apprepo.ErrVideoNotFound) {
			return dto.VideoLikeResponse{}, NotFoundError("video not found")
		}
		return dto.VideoLikeResponse{}, err
	}

	return dto.VideoLikeResponse{
		VideoID:   result.VideoID,
		HasLiked:  result.HasLiked,
		LikeCount: result.LikeCount,
	}, nil
}
