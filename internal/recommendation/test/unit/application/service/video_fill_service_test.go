package service_test

import (
	"context"
	"fmt"
	"testing"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

func TestDefaultVideoFillServiceSkipsRepositoryWhenSelectionAlreadyFull(t *testing.T) {
	reader := &spyVideoFillCandidateReader{}
	service := appservice.NewDefaultVideoFillService(reader)

	selected := []model.VideoCandidate{
		fillTestVideo("video-1", string(policy.LaneExactCore)),
		fillTestVideo("video-2", string(policy.LaneExactCore)),
	}
	filled, err := service.Fill(context.Background(), fillContext(2), selected, 2)
	if err != nil {
		t.Fatalf("Fill() error = %v", err)
	}

	if len(filled) != 2 {
		t.Fatalf("filled count = %d, want 2", len(filled))
	}
	if reader.masteredCalls != 0 || reader.popularCalls != 0 {
		t.Fatalf("repository calls = mastered:%d popular:%d, want none", reader.masteredCalls, reader.popularCalls)
	}
}

func TestDefaultVideoFillServiceFillsMasteredTargetBeforePopular(t *testing.T) {
	reader := &spyVideoFillCandidateReader{
		mastered: []model.VideoFillCandidate{
			fillCandidate("video-mastered-a", 120_000, 3, 10, 20, 3),
			fillCandidate("video-mastered-b", 110_000, 2, 8, 10, 2),
		},
		popular: []model.VideoFillCandidate{
			fillCandidate("video-popular-a", 100_000, 0, 0, 90, 30),
		},
	}
	service := appservice.NewDefaultVideoFillService(reader)

	filled, err := service.Fill(context.Background(), fillContext(4), []model.VideoCandidate{
		fillTestVideo("video-learning", string(policy.LaneExactCore)),
	}, 4)
	if err != nil {
		t.Fatalf("Fill() error = %v", err)
	}

	got := videoIDs(filled)
	want := []string{"video-learning", "video-mastered-a", "video-mastered-b", "video-popular-a"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("video ids = %v, want %v", got, want)
	}
	if len(filled[1].LearningUnits) != 0 || len(filled[2].LearningUnits) != 0 || len(filled[3].LearningUnits) != 0 {
		t.Fatalf("fill videos should not carry learning units: %#v", filled)
	}
	if filled[1].LaneSources[0] != string(policy.LaneMasteredTargetFill) {
		t.Fatalf("first fill lane = %v, want mastered target", filled[1].LaneSources)
	}
	if filled[3].LaneSources[0] != string(policy.LanePopularFill) {
		t.Fatalf("last fill lane = %v, want popular", filled[3].LaneSources)
	}
	if reader.masteredLimit != 9 {
		t.Fatalf("mastered query limit = %d, want 9", reader.masteredLimit)
	}
	if reader.popularLimit != 3 {
		t.Fatalf("popular query limit = %d, want 3", reader.popularLimit)
	}
	if !containsString(reader.popularExcluded, "video-mastered-a") || !containsString(reader.popularExcluded, "video-mastered-b") {
		t.Fatalf("popular excluded ids = %v, want selected and mastered fills excluded", reader.popularExcluded)
	}
}

func TestDefaultVideoFillServiceFillsPopularCandidatesToTarget(t *testing.T) {
	reader := &spyVideoFillCandidateReader{
		popular: []model.VideoFillCandidate{
			fillCandidate("video-popular-a", 100_000, 0, 0, 90, 30),
			fillCandidate("video-popular-b", 110_000, 0, 0, 80, 20),
			fillCandidate("video-popular-c", 120_000, 0, 0, 70, 10),
		},
	}
	service := appservice.NewDefaultVideoFillService(reader)

	filled, err := service.Fill(context.Background(), fillContext(4), []model.VideoCandidate{
		fillTestVideo("video-learning", string(policy.LaneExactCore)),
	}, 4)
	if err != nil {
		t.Fatalf("Fill() error = %v", err)
	}

	got := videoIDs(filled)
	want := []string{"video-learning", "video-popular-a", "video-popular-b", "video-popular-c"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("video ids = %v, want %v", got, want)
	}
	if len(filled) != 4 {
		t.Fatalf("filled count = %d, want target count 4", len(filled))
	}
}

func TestDefaultVideoFillServiceAllowsUnderfillWhenCandidatesAreInsufficient(t *testing.T) {
	reader := &spyVideoFillCandidateReader{
		popular: []model.VideoFillCandidate{
			fillCandidate("video-popular-a", 100_000, 0, 0, 90, 30),
		},
	}
	service := appservice.NewDefaultVideoFillService(reader)

	filled, err := service.Fill(context.Background(), fillContext(4), []model.VideoCandidate{
		fillTestVideo("video-learning", string(policy.LaneExactCore)),
	}, 4)
	if err != nil {
		t.Fatalf("Fill() error = %v", err)
	}

	got := videoIDs(filled)
	want := []string{"video-learning", "video-popular-a"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("video ids = %v, want %v", got, want)
	}
	if len(filled) != 2 {
		t.Fatalf("filled count = %d, want 2 because fill supply is insufficient", len(filled))
	}
}

func fillContext(targetCount int) model.RecommendationContext {
	return model.RecommendationContext{
		Request:              model.RecommendationRequest{UserID: "user-1", TargetVideoCount: targetCount},
		PreferredDurationSec: [2]int{45, 200},
	}
}

func fillTestVideo(videoID string, lane string) model.VideoCandidate {
	return model.VideoCandidate{
		VideoID:       videoID,
		DurationMs:    90_000,
		LaneSources:   []string{lane},
		LearningUnits: []model.ExpectedLearningUnit{{CoarseUnitID: 101}},
	}
}

func fillCandidate(videoID string, durationMs int32, matchedUnits int32, mentions int64, views int64, likes int64) model.VideoFillCandidate {
	return model.VideoFillCandidate{
		VideoID:           videoID,
		DurationMs:        durationMs,
		MatchedUnitCount:  matchedUnits,
		TotalMentionCount: mentions,
		MaxCoverageRatio:  0.25,
		MappedSpanRatio:   0.80,
		ViewCount:         views,
		LikeCount:         likes,
		FavoriteCount:     likes / 2,
	}
}

func videoIDs(videos []model.VideoCandidate) []string {
	result := make([]string, 0, len(videos))
	for _, video := range videos {
		result = append(result, video.VideoID)
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type spyVideoFillCandidateReader struct {
	mastered []model.VideoFillCandidate
	popular  []model.VideoFillCandidate

	masteredCalls   int
	popularCalls    int
	masteredLimit   int32
	popularLimit    int32
	popularExcluded []string
}

func (r *spyVideoFillCandidateReader) ListMasteredTargetFillCandidates(_ context.Context, _ string, _ []string, limit int32) ([]model.VideoFillCandidate, error) {
	r.masteredCalls++
	r.masteredLimit = limit
	return append([]model.VideoFillCandidate(nil), r.mastered...), nil
}

func (r *spyVideoFillCandidateReader) ListPopularFillCandidates(_ context.Context, _ string, excludedVideoIDs []string, limit int32) ([]model.VideoFillCandidate, error) {
	r.popularCalls++
	r.popularLimit = limit
	r.popularExcluded = append([]string(nil), excludedVideoIDs...)
	return append([]model.VideoFillCandidate(nil), r.popular...), nil
}

var _ apprepo.VideoFillCandidateReader = (*spyVideoFillCandidateReader)(nil)
