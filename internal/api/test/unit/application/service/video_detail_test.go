package service_test

import (
	"context"
	"testing"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

func TestVideoDetailServiceBuildsPublicDetailResponse(t *testing.T) {
	lookup := &fakeVideoDetailLookup{
		response: catalogdto.VideoDetailResponse{
			VideoID:              "11111111-1111-1111-1111-111111111111",
			Title:                "Visible title",
			Description:          "Visible description",
			VideoObjectPath:      "hls/111/master.m3u8",
			CoverImageURL:        stringPtr("covers/111.webp"),
			TranscriptObjectPath: stringPtr("transcripts/111.json"),
			DurationMS:           90500,
			ViewCount:            12,
			LikeCount:            3,
			FavoriteCount:        2,
			HasLiked:             true,
			HasFavorited:         false,
		},
	}
	service := apiservice.NewVideoDetailService(lookup, apiservice.NewPublicAssetURLBuilder("https://cdn.example.com/assets"))

	response, err := service.Execute(context.Background(), apvdto.GetVideoDetailRequest{
		UserID:  "user-1",
		VideoID: "11111111-1111-1111-1111-111111111111",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if lookup.request.UserID != "user-1" || lookup.request.VideoID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("lookup request = %+v", lookup.request)
	}
	if response.VideoURL != "https://cdn.example.com/assets/hls/111/master.m3u8" {
		t.Fatalf("video_url = %q", response.VideoURL)
	}
	if response.CoverImageURL == nil || *response.CoverImageURL != "https://cdn.example.com/assets/covers/111.webp" {
		t.Fatalf("cover_image_url = %+v", response.CoverImageURL)
	}
	if response.TranscriptURL == nil || *response.TranscriptURL != "https://cdn.example.com/assets/transcripts/111.json" {
		t.Fatalf("transcript_url = %+v", response.TranscriptURL)
	}
	if response.DurationSeconds != 91 || response.ViewCount != 12 || response.LikeCount != 3 || response.FavoriteCount != 2 {
		t.Fatalf("unexpected counts/duration: %+v", response)
	}
	if !response.UserState.HasLiked || response.UserState.HasFavorited {
		t.Fatalf("unexpected user_state: %+v", response.UserState)
	}
}

func TestVideoDetailServiceAllowsMissingTranscript(t *testing.T) {
	lookup := &fakeVideoDetailLookup{
		response: catalogdto.VideoDetailResponse{
			VideoID:         "22222222-2222-2222-2222-222222222222",
			Title:           "No transcript",
			VideoObjectPath: "https://cdn.example.com/hls/222/master.m3u8",
			DurationMS:      61000,
		},
	}
	service := apiservice.NewVideoDetailService(lookup, apiservice.NewPublicAssetURLBuilder("https://cdn.example.com/assets"))

	response, err := service.Execute(context.Background(), apvdto.GetVideoDetailRequest{
		UserID:  "user-1",
		VideoID: "22222222-2222-2222-2222-222222222222",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.TranscriptURL != nil {
		t.Fatalf("transcript_url = %+v, want nil", response.TranscriptURL)
	}
}

func TestVideoDetailServiceValidatesRequiredInputs(t *testing.T) {
	service := apiservice.NewVideoDetailService(&fakeVideoDetailLookup{}, apiservice.NewPublicAssetURLBuilder("https://cdn.example.com/assets"))

	for _, request := range []apvdto.GetVideoDetailRequest{
		{VideoID: "11111111-1111-1111-1111-111111111111"},
		{UserID: "user-1"},
	} {
		_, err := service.Execute(context.Background(), request)
		if err == nil {
			t.Fatalf("expected validation error for request %+v", request)
		}
		if !apiservice.IsInvalidRequest(err) {
			t.Fatalf("error = %v, want invalid request", err)
		}
	}
}

type fakeVideoDetailLookup struct {
	request  catalogdto.GetVideoDetailRequest
	response catalogdto.VideoDetailResponse
	err      error
}

func (f *fakeVideoDetailLookup) Execute(ctx context.Context, request catalogdto.GetVideoDetailRequest) (catalogdto.VideoDetailResponse, error) {
	f.request = request
	if f.err != nil {
		return catalogdto.VideoDetailResponse{}, f.err
	}
	return f.response, nil
}
