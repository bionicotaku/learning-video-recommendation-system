package service

import (
	"context"
	"fmt"
	"strings"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogusecase "learning-video-recommendation-system/internal/catalog/application/usecase"
)

type VideoDetailService struct {
	lookup     catalogusecase.GetVideoDetailUsecase
	urlBuilder PublicAssetURLBuilder
}

func NewVideoDetailService(lookup catalogusecase.GetVideoDetailUsecase, urlBuilder PublicAssetURLBuilder) *VideoDetailService {
	return &VideoDetailService{lookup: lookup, urlBuilder: urlBuilder}
}

func (s *VideoDetailService) Execute(ctx context.Context, request apvdto.GetVideoDetailRequest) (apvdto.VideoDetailResponse, error) {
	if request.UserID == "" {
		return apvdto.VideoDetailResponse{}, InvalidRequestError("user_id is required")
	}
	if request.VideoID == "" {
		return apvdto.VideoDetailResponse{}, InvalidRequestError("video_id is required")
	}
	if s.lookup == nil {
		return apvdto.VideoDetailResponse{}, fmt.Errorf("video detail lookup usecase is required")
	}

	detail, err := s.lookup.Execute(ctx, catalogdto.GetVideoDetailRequest{
		UserID:  request.UserID,
		VideoID: request.VideoID,
	})
	if err != nil {
		return apvdto.VideoDetailResponse{}, err
	}

	videoURL, err := s.urlBuilder.Build(detail.VideoObjectPath)
	if err != nil {
		return apvdto.VideoDetailResponse{}, fmt.Errorf("build video_url for video detail: video_id=%s: %w", request.VideoID, err)
	}
	coverURL, err := optionalPublicAssetURL(s.urlBuilder, detail.CoverImageURL)
	if err != nil {
		return apvdto.VideoDetailResponse{}, fmt.Errorf("build cover_image_url for video detail: video_id=%s: %w", request.VideoID, err)
	}
	transcriptURL, err := optionalPublicAssetURL(s.urlBuilder, detail.TranscriptObjectPath)
	if err != nil {
		return apvdto.VideoDetailResponse{}, fmt.Errorf("build transcript_url for video detail: video_id=%s: %w", request.VideoID, err)
	}

	return apvdto.VideoDetailResponse{
		VideoID:         detail.VideoID,
		Title:           detail.Title,
		Description:     detail.Description,
		VideoURL:        videoURL,
		CoverImageURL:   coverURL,
		TranscriptURL:   transcriptURL,
		DurationSeconds: durationSeconds(detail.DurationMS),
		ViewCount:       detail.ViewCount,
		LikeCount:       detail.LikeCount,
		FavoriteCount:   detail.FavoriteCount,
		UserState: apvdto.VideoDetailUserState{
			HasLiked:     detail.HasLiked,
			HasFavorited: detail.HasFavorited,
		},
	}, nil
}

func optionalPublicAssetURL(builder PublicAssetURLBuilder, path *string) (*string, error) {
	if path == nil || strings.TrimSpace(*path) == "" {
		return nil, nil
	}
	value, err := builder.Build(*path)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
