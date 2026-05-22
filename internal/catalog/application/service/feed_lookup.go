package service

import (
	"context"
	"errors"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	"learning-video-recommendation-system/internal/catalog/application/repository"
)

type FeedVideoLookupUsecase struct {
	reader repository.FeedVideoReader
}

func NewFeedVideoLookupUsecase(reader repository.FeedVideoReader) *FeedVideoLookupUsecase {
	return &FeedVideoLookupUsecase{reader: reader}
}

func (u *FeedVideoLookupUsecase) Execute(ctx context.Context, request dto.FeedVideoLookupRequest) (dto.FeedVideoLookupResponse, error) {
	if u.reader == nil {
		return dto.FeedVideoLookupResponse{}, errors.New("feed video reader is required")
	}
	if len(request.VideoIDs) == 0 {
		return dto.FeedVideoLookupResponse{}, nil
	}
	videos, err := u.reader.ListFeedVideosByIDs(ctx, request.UserID, request.VideoIDs)
	if err != nil {
		return dto.FeedVideoLookupResponse{}, err
	}
	result := make([]dto.FeedVideoDisplay, 0, len(videos))
	for _, video := range videos {
		result = append(result, dto.FeedVideoDisplay{
			VideoID:       video.VideoID,
			Title:         video.Title,
			CoverImageURL: video.CoverImageURL,
			ViewCount:     video.ViewCount,
		})
	}
	return dto.FeedVideoLookupResponse{Videos: result}, nil
}

type GetVideoDetailUsecase struct {
	reader repository.VideoDetailReader
}

func NewGetVideoDetailUsecase(reader repository.VideoDetailReader) *GetVideoDetailUsecase {
	return &GetVideoDetailUsecase{reader: reader}
}

func (u *GetVideoDetailUsecase) Execute(ctx context.Context, request dto.GetVideoDetailRequest) (dto.VideoDetailResponse, error) {
	if u.reader == nil {
		return dto.VideoDetailResponse{}, errors.New("video detail reader is required")
	}
	if request.UserID == "" {
		return dto.VideoDetailResponse{}, validationError("user_id is required")
	}
	if request.VideoID == "" {
		return dto.VideoDetailResponse{}, validationError("video_id is required")
	}
	detail, err := u.reader.GetVideoDetailByID(ctx, request.UserID, request.VideoID)
	if err != nil {
		if errors.Is(err, repository.ErrVideoNotFound) {
			return dto.VideoDetailResponse{}, NotFoundError("video not found")
		}
		return dto.VideoDetailResponse{}, err
	}
	return dto.VideoDetailResponse{
		VideoID:              detail.VideoID,
		Title:                detail.Title,
		Description:          detail.Description,
		VideoObjectPath:      detail.VideoObjectPath,
		CoverImageURL:        detail.CoverImageURL,
		TranscriptObjectPath: detail.TranscriptObjectPath,
		DurationMS:           detail.DurationMS,
		ViewCount:            detail.ViewCount,
		LikeCount:            detail.LikeCount,
		FavoriteCount:        detail.FavoriteCount,
		HasLiked:             detail.HasLiked,
		HasFavorited:         detail.HasFavorited,
	}, nil
}

type UnitLabelLookupUsecase struct {
	reader repository.UnitLabelReader
}

func NewUnitLabelLookupUsecase(reader repository.UnitLabelReader) *UnitLabelLookupUsecase {
	return &UnitLabelLookupUsecase{reader: reader}
}

func (u *UnitLabelLookupUsecase) Execute(ctx context.Context, request dto.UnitLabelLookupRequest) (dto.UnitLabelLookupResponse, error) {
	if u.reader == nil {
		return dto.UnitLabelLookupResponse{}, errors.New("unit label reader is required")
	}
	if len(request.CoarseUnitIDs) == 0 {
		return dto.UnitLabelLookupResponse{}, nil
	}
	labels, err := u.reader.ListUnitLabelsByIDs(ctx, request.CoarseUnitIDs)
	if err != nil {
		return dto.UnitLabelLookupResponse{}, err
	}
	result := make([]dto.UnitLabel, 0, len(labels))
	for _, label := range labels {
		result = append(result, dto.UnitLabel{
			CoarseUnitID: label.CoarseUnitID,
			Text:         label.Text,
		})
	}
	return dto.UnitLabelLookupResponse{Labels: result}, nil
}
