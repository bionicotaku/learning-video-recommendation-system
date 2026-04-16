package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	recommendationservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

type stubRecommendableVideoUnitReader struct {
	rows    []model.RecommendableVideoUnit
	lastCtx context.Context
	err     error
}

var _ apprepo.RecommendableVideoUnitReader = (*stubRecommendableVideoUnitReader)(nil)

func (s *stubRecommendableVideoUnitReader) ListByUnitIDs(ctx context.Context, coarseUnitIDs []int64) ([]model.RecommendableVideoUnit, error) {
	s.lastCtx = ctx
	if s.err != nil {
		return nil, s.err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	allowed := make(map[int64]struct{}, len(coarseUnitIDs))
	for _, id := range coarseUnitIDs {
		allowed[id] = struct{}{}
	}

	result := make([]model.RecommendableVideoUnit, 0, len(s.rows))
	for _, row := range s.rows {
		if _, ok := allowed[row.CoarseUnitID]; ok {
			result = append(result, row)
		}
	}
	return result, nil
}

func TestDefaultCandidateGeneratorExactCoreRanksCoreHitsByStrength(t *testing.T) {
	generator := recommendationservice.NewDefaultCandidateGenerator(&stubRecommendableVideoUnitReader{
		rows: []model.RecommendableVideoUnit{
			recommendableRow("video-hard-strong", 101, 4, 3, 42, 0.52, 0.91, 120_000),
			recommendableRow("video-hard-weak", 101, 1, 1, 12, 0.12, 0.31, 120_000),
			recommendableRow("video-new-top", 201, 3, 2, 31, 0.33, 0.70, 95_000),
			recommendableRow("video-new-low", 202, 2, 1, 18, 0.21, 0.52, 90_000),
		},
	})

	candidates, err := generator.Generate(context.Background(), recommendationContext(), recommendationDemand())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	exact := filterCandidatesByLane(candidates, string(policy.LaneExactCore))
	if len(exact) < 2 {
		t.Fatalf("expected exact_core candidates, got %#v", exact)
	}
	if exact[0].VideoID != "video-hard-strong" {
		t.Fatalf("expected strongest hard review video first, got %#v", exact[0])
	}
	if !containsVideo(exact, "video-new-top") {
		t.Fatalf("expected top-ranked new_now video in exact_core lane, got %#v", exact)
	}
	if containsVideo(exact, "video-new-low") {
		t.Fatalf("did not expect lower-ranked new_now video in exact_core lane, got %#v", exact)
	}
}

func TestDefaultCandidateGeneratorBundleRequiresTwoUnitsAndCoreCoverageByDefault(t *testing.T) {
	generator := recommendationservice.NewDefaultCandidateGenerator(&stubRecommendableVideoUnitReader{
		rows: []model.RecommendableVideoUnit{
			recommendableRow("video-bundle", 101, 3, 2, 28, 0.24, 0.65, 110_000),
			recommendableRow("video-bundle", 301, 2, 2, 20, 0.18, 0.62, 110_000),
			recommendableRow("video-single-core", 101, 4, 3, 36, 0.41, 0.88, 105_000),
			recommendableRow("video-soft-only", 301, 2, 2, 20, 0.18, 0.62, 105_000),
			recommendableRow("video-soft-only", 302, 2, 2, 20, 0.18, 0.62, 105_000),
		},
	})

	candidates, err := generator.Generate(context.Background(), recommendationContext(), recommendationDemand())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	bundle := filterCandidatesByLane(candidates, string(policy.LaneBundle))
	if !containsVideo(bundle, "video-bundle") {
		t.Fatalf("expected bundle-qualified video, got %#v", bundle)
	}
	if containsVideo(bundle, "video-single-core") {
		t.Fatalf("single-unit core video must not enter bundle lane, got %#v", bundle)
	}
	if containsVideo(bundle, "video-soft-only") {
		t.Fatalf("soft-only bundle should not pass without low-supply relaxation, got %#v", bundle)
	}
}

func TestDefaultCandidateGeneratorBundleRelaxesWhenHardReviewSupplyIsLow(t *testing.T) {
	recommendationCtx := recommendationContext()
	demand := recommendationDemand()
	demand.Flags.HardReviewLowSupply = true

	generator := recommendationservice.NewDefaultCandidateGenerator(&stubRecommendableVideoUnitReader{
		rows: []model.RecommendableVideoUnit{
			recommendableRow("video-soft-relaxed", 301, 3, 2, 24, 0.22, 0.71, 115_000),
			recommendableRow("video-soft-relaxed", 302, 3, 2, 22, 0.20, 0.68, 115_000),
		},
	})

	candidates, err := generator.Generate(context.Background(), recommendationCtx, demand)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	bundle := filterCandidatesByLane(candidates, string(policy.LaneBundle))
	if !containsVideo(bundle, "video-soft-relaxed") {
		t.Fatalf("expected relaxed soft bundle candidate under low supply, got %#v", bundle)
	}
}

func TestDefaultCandidateGeneratorSoftFutureUsesOnlySoftAndNearFuture(t *testing.T) {
	generator := recommendationservice.NewDefaultCandidateGenerator(&stubRecommendableVideoUnitReader{
		rows: []model.RecommendableVideoUnit{
			recommendableRow("video-soft", 301, 2, 2, 22, 0.23, 0.66, 100_000),
			recommendableRow("video-future", 401, 2, 2, 18, 0.17, 0.63, 100_000),
			recommendableRow("video-core-ignored", 101, 4, 3, 36, 0.42, 0.91, 100_000),
		},
	})

	candidates, err := generator.Generate(context.Background(), recommendationContext(), recommendationDemand())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	softFuture := filterCandidatesByLane(candidates, string(policy.LaneSoftFuture))
	if !containsVideo(softFuture, "video-soft") || !containsVideo(softFuture, "video-future") {
		t.Fatalf("expected soft_review and near_future candidates, got %#v", softFuture)
	}
	if containsVideo(softFuture, "video-core-ignored") {
		t.Fatalf("hard_review units must not appear in soft_future lane, got %#v", softFuture)
	}
}

func TestDefaultCandidateGeneratorQualityFallbackOnlyFillsRemainingGap(t *testing.T) {
	generator := recommendationservice.NewDefaultCandidateGenerator(&stubRecommendableVideoUnitReader{
		rows: []model.RecommendableVideoUnit{
			recommendableRow("video-hard", 101, 4, 3, 40, 0.51, 0.90, 110_000),
			recommendableRow("video-soft", 301, 2, 2, 20, 0.19, 0.66, 100_000),
			recommendableRow("video-future", 401, 2, 2, 18, 0.16, 0.61, 100_000),
			recommendableRow("video-fallback-best", 202, 1, 1, 12, 0.08, 0.55, 95_000),
			recommendableRow("video-fallback-worse", 202, 1, 1, 10, 0.05, 0.40, 260_000),
		},
	})

	candidates, err := generator.Generate(context.Background(), recommendationContext(), recommendationDemand())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	fallback := filterCandidatesByLane(candidates, string(policy.LaneQualityFallback))
	if len(fallback) != 1 {
		t.Fatalf("expected exactly one fallback candidate, got %#v", fallback)
	}
	if fallback[0].VideoID != "video-fallback-best" {
		t.Fatalf("expected strongest fallback candidate, got %#v", fallback[0])
	}
}

func TestDefaultCandidateGeneratorScenarioCoreStillLeadsWhenNearFutureInventoryIsLarge(t *testing.T) {
	rows := []model.RecommendableVideoUnit{
		recommendableRow("video-hard-main", 101, 4, 3, 38, 0.49, 0.88, 110_000),
		recommendableRow("video-bundle-main", 101, 3, 2, 26, 0.27, 0.71, 105_000),
		recommendableRow("video-bundle-main", 301, 3, 2, 24, 0.24, 0.69, 105_000),
	}
	for i := 0; i < 6; i++ {
		rows = append(rows, recommendableRow(videoID("video-future-", i), 401, 2, 2, 16, 0.15, 0.60, 100_000))
	}

	generator := recommendationservice.NewDefaultCandidateGenerator(&stubRecommendableVideoUnitReader{rows: rows})
	candidates, err := generator.Generate(context.Background(), recommendationContext(), recommendationDemand())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	summary := summarizeCandidates(candidates)
	if summary.LaneDistinctVideos[string(policy.LaneExactCore)] == 0 {
		t.Fatalf("expected exact_core candidates in normal scenario, got %#v", summary)
	}
	if summary.LaneDistinctVideos[string(policy.LaneBundle)] == 0 {
		t.Fatalf("expected bundle candidates in normal scenario, got %#v", summary)
	}
	if summary.OrderedVideos[0] != "video-hard-main" {
		t.Fatalf("expected core candidate to lead ordered videos, got %#v", summary.OrderedVideos)
	}
}

func TestDefaultCandidateGeneratorScenarioGoldenCandidateSummary(t *testing.T) {
	recommendationCtx := recommendationContext()
	demand := recommendationDemand()
	demand.Flags.HardReviewLowSupply = true

	rows := []model.RecommendableVideoUnit{
		recommendableRow("video-exact-a", 101, 4, 3, 42, 0.52, 0.92, 120_000),
		recommendableRow("video-exact-b", 201, 3, 2, 30, 0.31, 0.72, 95_000),
		recommendableRow("video-bundle-a", 101, 2, 2, 20, 0.20, 0.65, 110_000),
		recommendableRow("video-bundle-a", 301, 2, 2, 18, 0.17, 0.62, 110_000),
		recommendableRow("video-soft-a", 301, 2, 2, 21, 0.19, 0.68, 105_000),
		recommendableRow("video-soft-b", 401, 2, 2, 16, 0.14, 0.61, 105_000),
		recommendableRow("video-fallback", 402, 1, 1, 10, 0.07, 0.44, 90_000),
	}

	generator := recommendationservice.NewDefaultCandidateGenerator(&stubRecommendableVideoUnitReader{rows: rows})
	candidates, err := generator.Generate(context.Background(), recommendationCtx, demand)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	actual, err := json.MarshalIndent(summarizeCandidates(candidates), "", "  ")
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	goldenPath := filepath.Join(filepath.Dir(currentFile), "../../../golden/candidate_summary_review_sparse.json")
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
		t.Fatalf("candidate summary golden mismatch\nactual:\n%s\nexpected:\n%s", actual, expected)
	}
}

func TestDefaultCandidateGeneratorPropagatesContextToReader(t *testing.T) {
	reader := &stubRecommendableVideoUnitReader{}
	generator := recommendationservice.NewDefaultCandidateGenerator(reader)

	ctx := context.WithValue(context.Background(), "trace", "candidate-generator")
	_, err := generator.Generate(ctx, recommendationContext(), recommendationDemand())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if got := reader.lastCtx.Value("trace"); got != "candidate-generator" {
		t.Fatalf("reader ctx value = %#v, want propagated request context", got)
	}
}

func TestDefaultCandidateGeneratorReturnsCanceledWhenContextCanceled(t *testing.T) {
	reader := &stubRecommendableVideoUnitReader{}
	generator := recommendationservice.NewDefaultCandidateGenerator(reader)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := generator.Generate(ctx, recommendationContext(), recommendationDemand())
	if err == nil || err != context.Canceled {
		t.Fatalf("generate error = %v, want context.Canceled", err)
	}
}

type candidateSummary struct {
	LaneCounts         map[string]int      `json:"lane_counts"`
	LaneDistinctVideos map[string]int      `json:"lane_distinct_videos"`
	DistinctVideoCount int                 `json:"distinct_video_count"`
	OrderedVideos      []string            `json:"ordered_videos"`
	VideoLanes         map[string][]string `json:"video_lanes"`
}

func summarizeCandidates(candidates []model.VideoUnitCandidate) candidateSummary {
	summary := candidateSummary{
		LaneCounts:         map[string]int{},
		LaneDistinctVideos: map[string]int{},
		VideoLanes:         map[string][]string{},
	}

	distinctVideos := make(map[string]struct{})
	laneVideos := make(map[string]map[string]struct{})
	for _, candidate := range candidates {
		summary.LaneCounts[candidate.Lane]++
		distinctVideos[candidate.VideoID] = struct{}{}
		if _, ok := laneVideos[candidate.Lane]; !ok {
			laneVideos[candidate.Lane] = map[string]struct{}{}
		}
		laneVideos[candidate.Lane][candidate.VideoID] = struct{}{}
		summary.VideoLanes[candidate.VideoID] = appendUnique(summary.VideoLanes[candidate.VideoID], candidate.Lane)
	}

	for lane, videos := range laneVideos {
		summary.LaneDistinctVideos[lane] = len(videos)
	}

	summary.DistinctVideoCount = len(distinctVideos)
	summary.OrderedVideos = orderedDistinctVideos(candidates)
	for videoID := range summary.VideoLanes {
		sort.Strings(summary.VideoLanes[videoID])
	}

	return summary
}

func orderedDistinctVideos(candidates []model.VideoUnitCandidate) []string {
	ordered := make([]string, 0)
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate.VideoID]; ok {
			continue
		}
		seen[candidate.VideoID] = struct{}{}
		ordered = append(ordered, candidate.VideoID)
	}
	return ordered
}

func filterCandidatesByLane(candidates []model.VideoUnitCandidate, lane string) []model.VideoUnitCandidate {
	result := make([]model.VideoUnitCandidate, 0)
	for _, candidate := range candidates {
		if candidate.Lane == lane {
			result = append(result, candidate)
		}
	}
	return result
}

func containsVideo(candidates []model.VideoUnitCandidate, videoID string) bool {
	for _, candidate := range candidates {
		if candidate.VideoID == videoID {
			return true
		}
	}
	return false
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func recommendableRow(videoID string, unitID int64, mentions int32, sentences int32, coverageMs int32, coverageRatio float64, mappedSpanRatio float64, durationMs int32) model.RecommendableVideoUnit {
	return model.RecommendableVideoUnit{
		VideoID:          videoID,
		CoarseUnitID:     unitID,
		MentionCount:     mentions,
		SentenceCount:    sentences,
		CoverageMs:       coverageMs,
		CoverageRatio:    coverageRatio,
		SentenceIndexes:  []int32{1, 2},
		DurationMs:       durationMs,
		MappedSpanRatio:  mappedSpanRatio,
		EvidenceSpanRefs: []byte(`[{"sentence_index":1,"span_index":1}]`),
	}
}

func recommendationContext() model.RecommendationContext {
	return model.RecommendationContext{
		Request: model.RecommendationRequest{
			UserID:               "user-1",
			TargetVideoCount:     4,
			PreferredDurationSec: [2]int{45, 180},
		},
	}
}

func recommendationDemand() model.DemandBundle {
	return model.DemandBundle{
		HardReview: []model.DemandUnit{
			{UnitID: 101, Bucket: string(policy.BucketHardReview), Weight: 1.0, SupplyGrade: "ok"},
		},
		NewNow: []model.DemandUnit{
			{UnitID: 201, Bucket: string(policy.BucketNewNow), Weight: 0.9, SupplyGrade: "ok"},
			{UnitID: 202, Bucket: string(policy.BucketNewNow), Weight: 0.4, SupplyGrade: "weak"},
		},
		SoftReview: []model.DemandUnit{
			{UnitID: 301, Bucket: string(policy.BucketSoftReview), Weight: 0.6, SupplyGrade: "strong"},
			{UnitID: 302, Bucket: string(policy.BucketSoftReview), Weight: 0.55, SupplyGrade: "strong"},
		},
		NearFuture: []model.DemandUnit{
			{UnitID: 401, Bucket: string(policy.BucketNearFuture), Weight: 0.5, SupplyGrade: "strong"},
			{UnitID: 402, Bucket: string(policy.BucketNearFuture), Weight: 0.45, SupplyGrade: "ok"},
		},
		TargetVideoCount: 4,
		LaneBudget: model.LaneBudget{
			ExactCore:       0.45,
			Bundle:          0.30,
			SoftFuture:      0.15,
			QualityFallback: 0.10,
		},
	}
}

func videoID(prefix string, index int) string {
	return prefix + string(rune('a'+index))
}
