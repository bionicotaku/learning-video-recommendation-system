package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	recommendationdto "learning-video-recommendation-system/internal/recommendation/application/dto"
)

func TestFeedServiceBuildsDisplayResponseFromRecommendationPlan(t *testing.T) {
	recommender := &fakeFeedRecommender{
		response: recommendationdto.GenerateVideoRecommendationsResponse{
			RunID: "run-1",
			Items: []recommendationdto.RecommendationPlanItem{
				{
					VideoID:    "11111111-1111-1111-1111-111111111111",
					DurationMs: 90500,
					LearningUnits: []recommendationdto.ExpectedLearningUnit{
						{
							CoarseUnitID: 101,
							Role:         "hard_review",
							IsPrimary:    true,
							Evidence: &recommendationdto.LearningUnitEvidence{
								SentenceIndex: int32ptr(2),
								SpanIndex:     int32ptr(1),
								StartMs:       int32ptr(2000),
								EndMs:         int32ptr(2400),
							},
						},
					},
				},
			},
		},
	}
	videoLookup := &fakeFeedVideoLookup{
		response: catalogdto.FeedVideoLookupResponse{Videos: []catalogdto.FeedVideoDisplay{
			{
				VideoID:         "11111111-1111-1111-1111-111111111111",
				Title:           "Title",
				Description:     "Description",
				VideoObjectPath: "hls/111/master.m3u8",
				CoverImageURL:   stringPtr("covers/111.webp"),
				ViewCount:       12,
				LikeCount:       3,
				FavoriteCount:   2,
			},
		}},
	}
	labelLookup := &fakeUnitLabelLookup{
		response: catalogdto.UnitLabelLookupResponse{Labels: []catalogdto.UnitLabel{{CoarseUnitID: 101, Text: "serendipity"}}},
	}
	service := apiservice.NewFeedService(recommender, videoLookup, labelLookup, apiservice.NewPublicAssetURLBuilder("https://cdn.example.com/assets"), discardLogger())

	response, err := service.Execute(context.Background(), apvdto.GetFeedRequest{
		UserID:               "user-1",
		TargetVideoCount:     8,
		PreferredDurationSec: [2]int{20, 120},
		SessionHint:          "mixed",
		ClientContext:        []byte(`{"platform":"ios"}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if recommender.request.UserID != "user-1" || recommender.request.TargetVideoCount != 8 {
		t.Fatalf("recommendation request not mapped: %+v", recommender.request)
	}
	if len(videoLookup.request.VideoIDs) != 1 || videoLookup.request.VideoIDs[0] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("video lookup request = %+v", videoLookup.request)
	}
	if len(labelLookup.request.CoarseUnitIDs) != 1 || labelLookup.request.CoarseUnitIDs[0] != 101 {
		t.Fatalf("unit lookup request = %+v", labelLookup.request)
	}

	if response.RecommendationRunID != "run-1" || len(response.Items) != 1 {
		t.Fatalf("unexpected response shell: %+v", response)
	}
	item := response.Items[0]
	if item.VideoID != "11111111-1111-1111-1111-111111111111" || item.VideoURL != "https://cdn.example.com/assets/hls/111/master.m3u8" {
		t.Fatalf("unexpected item identity/url: %+v", item)
	}
	if item.CoverImageURL == nil || *item.CoverImageURL != "https://cdn.example.com/assets/covers/111.webp" {
		t.Fatalf("unexpected cover url: %+v", item.CoverImageURL)
	}
	if item.DurationSeconds != 91 || item.ViewCount != 12 || item.LikeCount != 3 || item.FavoriteCount != 2 {
		t.Fatalf("unexpected counts/duration: %+v", item)
	}
	if len(item.LearningUnits) != 1 {
		t.Fatalf("expected 1 learning unit, got %+v", item.LearningUnits)
	}
	unit := item.LearningUnits[0]
	if unit.CoarseUnitID != 101 || unit.Text != "serendipity" || unit.Role != "hard_review" || !unit.IsPrimary {
		t.Fatalf("unexpected unit: %+v", unit)
	}
	if unit.EvidenceSentenceIndex != 2 || unit.EvidenceSpanIndex != 1 || unit.EvidenceStartMS != 2000 || unit.EvidenceEndMS != 2400 {
		t.Fatalf("unexpected evidence: %+v", unit)
	}
}

func TestFeedServicePropagatesRecommendationError(t *testing.T) {
	service := apiservice.NewFeedService(
		&fakeFeedRecommender{err: errors.New("recommendation down")},
		&fakeFeedVideoLookup{},
		&fakeUnitLabelLookup{},
		apiservice.NewPublicAssetURLBuilder("https://cdn.example.com"),
		discardLogger(),
	)

	_, err := service.Execute(context.Background(), apvdto.GetFeedRequest{UserID: "user-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFeedServiceFailsWhenPlanCannotBeFullyMaterialized(t *testing.T) {
	cases := []struct {
		name        string
		planItem    recommendationdto.RecommendationPlanItem
		videos      []catalogdto.FeedVideoDisplay
		labels      []catalogdto.UnitLabel
		errContains string
	}{
		{
			name:        "missing video display data",
			planItem:    validPlanItem("11111111-1111-1111-1111-111111111111", 101),
			videos:      nil,
			labels:      []catalogdto.UnitLabel{{CoarseUnitID: 101, Text: "kept"}},
			errContains: "missing feed video display data",
		},
		{
			name:        "invalid duration",
			planItem:    invalidDurationPlanItem("11111111-1111-1111-1111-111111111111", 101),
			videos:      []catalogdto.FeedVideoDisplay{validVideoDisplay("11111111-1111-1111-1111-111111111111")},
			labels:      []catalogdto.UnitLabel{{CoarseUnitID: 101, Text: "kept"}},
			errContains: "invalid duration_ms",
		},
		{
			name: "incomplete evidence",
			planItem: recommendationdto.RecommendationPlanItem{
				VideoID:       "11111111-1111-1111-1111-111111111111",
				DurationMs:    30000,
				LearningUnits: []recommendationdto.ExpectedLearningUnit{{CoarseUnitID: 101, Role: "hard_review"}},
			},
			videos:      []catalogdto.FeedVideoDisplay{validVideoDisplay("11111111-1111-1111-1111-111111111111")},
			labels:      []catalogdto.UnitLabel{{CoarseUnitID: 101, Text: "kept"}},
			errContains: "incomplete learning unit evidence",
		},
		{
			name:        "missing label",
			planItem:    validPlanItem("11111111-1111-1111-1111-111111111111", 101),
			videos:      []catalogdto.FeedVideoDisplay{validVideoDisplay("11111111-1111-1111-1111-111111111111")},
			labels:      nil,
			errContains: "missing unit label",
		},
		{
			name:        "invalid url",
			planItem:    validPlanItem("11111111-1111-1111-1111-111111111111", 101),
			videos:      []catalogdto.FeedVideoDisplay{{VideoID: "11111111-1111-1111-1111-111111111111", Title: "Title"}},
			labels:      []catalogdto.UnitLabel{{CoarseUnitID: 101, Text: "kept"}},
			errContains: "build video_url",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			service := apiservice.NewFeedService(
				&fakeFeedRecommender{response: recommendationdto.GenerateVideoRecommendationsResponse{RunID: "run-1", Items: []recommendationdto.RecommendationPlanItem{tt.planItem}}},
				&fakeFeedVideoLookup{response: catalogdto.FeedVideoLookupResponse{Videos: tt.videos}},
				&fakeUnitLabelLookup{response: catalogdto.UnitLabelLookupResponse{Labels: tt.labels}},
				apiservice.NewPublicAssetURLBuilder("https://cdn.example.com/assets"),
				discardLogger(),
			)

			_, err := service.Execute(context.Background(), apvdto.GetFeedRequest{UserID: "user-1"})
			if err == nil {
				t.Fatal("expected materialization error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error = %v, want containing %q", err, tt.errContains)
			}
		})
	}
}

func completeEvidence(startMS int32, endMS int32) *recommendationdto.LearningUnitEvidence {
	return &recommendationdto.LearningUnitEvidence{
		SentenceIndex: int32ptr(1),
		SpanIndex:     int32ptr(1),
		StartMs:       &startMS,
		EndMs:         &endMS,
	}
}

func validPlanItem(videoID string, unitID int64) recommendationdto.RecommendationPlanItem {
	return recommendationdto.RecommendationPlanItem{
		VideoID:       videoID,
		DurationMs:    30000,
		LearningUnits: []recommendationdto.ExpectedLearningUnit{{CoarseUnitID: unitID, Role: "hard_review", Evidence: completeEvidence(1000, 1200)}},
	}
}

func invalidDurationPlanItem(videoID string, unitID int64) recommendationdto.RecommendationPlanItem {
	item := validPlanItem(videoID, unitID)
	item.DurationMs = 0
	return item
}

func validVideoDisplay(videoID string) catalogdto.FeedVideoDisplay {
	return catalogdto.FeedVideoDisplay{
		VideoID:         videoID,
		Title:           "Title",
		VideoObjectPath: "https://cdn.example.com/hls/master.m3u8",
	}
}

func int32ptr(value int32) *int32 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

type fakeFeedRecommender struct {
	request  recommendationdto.GenerateVideoRecommendationsRequest
	response recommendationdto.GenerateVideoRecommendationsResponse
	err      error
}

func (f *fakeFeedRecommender) Execute(ctx context.Context, request recommendationdto.GenerateVideoRecommendationsRequest) (recommendationdto.GenerateVideoRecommendationsResponse, error) {
	f.request = request
	return f.response, f.err
}

type fakeFeedVideoLookup struct {
	request  catalogdto.FeedVideoLookupRequest
	response catalogdto.FeedVideoLookupResponse
	err      error
}

func (f *fakeFeedVideoLookup) Execute(ctx context.Context, request catalogdto.FeedVideoLookupRequest) (catalogdto.FeedVideoLookupResponse, error) {
	f.request = request
	return f.response, f.err
}

type fakeUnitLabelLookup struct {
	request  catalogdto.UnitLabelLookupRequest
	response catalogdto.UnitLabelLookupResponse
	err      error
}

func (f *fakeUnitLabelLookup) Execute(ctx context.Context, request catalogdto.UnitLabelLookupRequest) (catalogdto.UnitLabelLookupResponse, error) {
	f.request = request
	return f.response, f.err
}
