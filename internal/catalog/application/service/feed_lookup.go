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
			VideoID:              video.VideoID,
			Title:                video.Title,
			Description:          video.Description,
			VideoObjectPath:      video.VideoObjectPath,
			CoverImageURL:        video.CoverImageURL,
			TranscriptObjectPath: video.TranscriptObjectPath,
			ViewCount:            video.ViewCount,
			LikeCount:            video.LikeCount,
			FavoriteCount:        video.FavoriteCount,
			HasLiked:             video.HasLiked,
			HasFavorited:         video.HasFavorited,
		})
	}
	return dto.FeedVideoLookupResponse{Videos: result}, nil
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
