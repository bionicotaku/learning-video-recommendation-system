package usecase_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"learning-video-recommendation-system/internal/recommendation/application/dto"
	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/application/usecase"
	domainaggregator "learning-video-recommendation-system/internal/recommendation/domain/aggregator"
	domaincandidate "learning-video-recommendation-system/internal/recommendation/domain/candidate"
	domainexplain "learning-video-recommendation-system/internal/recommendation/domain/explain"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	domainplanner "learning-video-recommendation-system/internal/recommendation/domain/planner"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
	domainranking "learning-video-recommendation-system/internal/recommendation/domain/ranking"
	domainresolver "learning-video-recommendation-system/internal/recommendation/domain/resolver"
	domainselector "learning-video-recommendation-system/internal/recommendation/domain/selector"
)

func TestGenerateVideoRecommendationsPipelineExecutesFullRecommendationFlow(t *testing.T) {
	writer := &spyResultWriter{}
	ranker := &stubRanker{
		ranked: []model.VideoCandidate{
			testVideoCandidate("video-1", 101, model.LearningRoleHardReview),
		},
	}

	service, err := usecase.NewGenerateVideoRecommendationsPipeline(
		&constructorStubContextAssembler{
			context: model.RecommendationContext{
				PreferredDurationSec: [2]int{45, 200},
				Request:              model.RecommendationRequest{UserID: "user-1", TargetVideoCount: 2},
				RecallScope: model.RecallScopeSummary{
					ActiveTargetUnitCount:         12,
					QueueCandidateCount:           10,
					PlannerScopeUnitCount:         5,
					PlannerScopeUnitCountByBucket: map[string]int{"hard_review": 3, "new_now": 2},
					NoSupplyScopeUnitCount:        1,
					RecallFetchUnitCount:          4,
					PerUnitRecallLimit:            20,
					MaxPossibleRecallRows:         80,
					ActualRecallRowCount:          42,
					AggregatedVideoCandidateCount: 1,
					VideoStateLookupCount:         1,
				},
			},
		},
		&stubPlanner{
			bundle: model.DemandBundle{
				TargetVideoCount: 2,
				Flags:            model.PlannerFlags{HardReviewLowSupply: true},
			},
		},
		&stubCandidateGenerator{candidates: []model.VideoUnitCandidate{{VideoID: "video-1", CoarseUnitID: 101}}},
		&stubResolver{windows: []model.ResolvedEvidenceWindow{{Candidate: model.VideoUnitCandidate{VideoID: "video-1", CoarseUnitID: 101}}}},
		&stubAggregator{videos: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		ranker,
		&stubSelector{selected: ranker.ranked},
		&stubVideoFillService{},
		&stubExplainer{items: []model.FinalRecommendationItem{testFinalItem("video-1", 101, model.LearningRoleHardReview)}},
		&stubVideoStateEnricher{
			videoServingStates: []model.UserVideoServingState{{VideoID: "video-1"}},
			videoUserStates:    []model.VideoUserState{{VideoID: "video-1", WatchCount: 1}},
		},
		writer,
	)
	if err != nil {
		t.Fatalf("NewGenerateVideoRecommendationsPipeline() error = %v", err)
	}

	response, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{
		UserID:           "user-1",
		TargetVideoCount: 2,
	})
	if err != nil {
		t.Fatalf("execute pipeline: %v", err)
	}

	if response.RunID == "" {
		t.Fatal("expected generated run id")
	}
	if len(response.Items) != 1 || response.Items[0].VideoID != "video-1" {
		t.Fatalf("unexpected response items: %#v", response.Items)
	}
	if response.Items[0].DurationMs != 90_000 {
		t.Fatalf("duration_ms = %d, want 90000", response.Items[0].DurationMs)
	}
	if !writer.called {
		t.Fatal("expected result writer to persist outputs")
	}
	if writer.run.SelectorMode != "low_supply" {
		t.Fatalf("expected low_supply audit selector mode, got %q", writer.run.SelectorMode)
	}
	if ranker.lastContextVideoStates != 1 {
		t.Fatalf("expected ranker to receive loaded video serving states, got %d", ranker.lastContextVideoStates)
	}
	if ranker.lastContextUserStates != 1 {
		t.Fatalf("expected ranker to receive loaded video user states, got %d", ranker.lastContextUserStates)
	}

	var summary map[string]any
	if err := json.Unmarshal(writer.run.CandidateSummary, &summary); err != nil {
		t.Fatalf("unmarshal candidate summary: %v", err)
	}
	if summary["planner_scope_unit_count"] != float64(5) {
		t.Fatalf("planner scope count = %#v", summary["planner_scope_unit_count"])
	}
	if summary["no_supply_scope_unit_count"] != float64(1) {
		t.Fatalf("no-supply scope count = %#v", summary["no_supply_scope_unit_count"])
	}
	timing, ok := summary["pipeline_timing_ms"].(map[string]any)
	if !ok {
		t.Fatalf("pipeline_timing_ms missing or invalid: %#v", summary["pipeline_timing_ms"])
	}
	for _, key := range []string{"context_assemble", "plan", "candidate_generate", "evidence_resolve", "aggregate", "video_state_enrich", "rank", "select", "fill", "final_item_build", "total"} {
		value, ok := timing[key].(float64)
		if !ok || value < 0 {
			t.Fatalf("timing[%s] = %#v", key, timing[key])
		}
	}
	for _, key := range []string{"audit_write", "serving_state_write"} {
		if _, ok := timing[key]; ok {
			t.Fatalf("timing[%s] should not be present in audit summary: %#v", key, timing)
		}
	}
}

func TestGenerateVideoRecommendationsPipelineGoldenResponse(t *testing.T) {
	service, err := usecase.NewGenerateVideoRecommendationsPipeline(
		&constructorStubContextAssembler{
			context: model.RecommendationContext{
				PreferredDurationSec: [2]int{45, 200},
				Request:              model.RecommendationRequest{UserID: "user-1", TargetVideoCount: 2},
			},
		},
		&stubPlanner{bundle: model.DemandBundle{TargetVideoCount: 2}},
		&stubCandidateGenerator{candidates: []model.VideoUnitCandidate{{VideoID: "video-1", CoarseUnitID: 101}}},
		&stubResolver{windows: []model.ResolvedEvidenceWindow{{Candidate: model.VideoUnitCandidate{VideoID: "video-1", CoarseUnitID: 101}}}},
		&stubAggregator{videos: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		&stubRanker{ranked: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		&stubSelector{selected: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		&stubVideoFillService{},
		&stubExplainer{items: []model.FinalRecommendationItem{testFinalItem("video-1", 101, model.LearningRoleHardReview)}},
		&stubVideoStateEnricher{},
		&spyResultWriter{},
	)
	if err != nil {
		t.Fatalf("NewGenerateVideoRecommendationsPipeline() error = %v", err)
	}

	response, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{
		UserID:           "user-1",
		TargetVideoCount: 2,
	})
	if err != nil {
		t.Fatalf("execute pipeline: %v", err)
	}
	response.RunID = "fixed-run-id"

	actual, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	goldenPath := filepath.Join(filepath.Dir(currentFile), "../../../golden/usecase_pipeline_response.json")
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
		t.Fatalf("usecase response golden mismatch\nactual:\n%s\nexpected:\n%s", actual, expected)
	}
}

func TestGenerateVideoRecommendationsPipelineMarksExtremeSparseAfterSelectionUnderfill(t *testing.T) {
	writer := &spyResultWriter{}
	service, err := usecase.NewGenerateVideoRecommendationsPipeline(
		&constructorStubContextAssembler{
			context: model.RecommendationContext{
				PreferredDurationSec: [2]int{45, 200},
				Request:              model.RecommendationRequest{UserID: "user-1", TargetVideoCount: 3},
			},
		},
		&stubPlanner{
			bundle: model.DemandBundle{
				TargetVideoCount: 3,
				HardReview:       []model.DemandUnit{{UnitID: 101, Bucket: "hard_review"}},
			},
		},
		&stubCandidateGenerator{candidates: []model.VideoUnitCandidate{{VideoID: "video-1", CoarseUnitID: 101}}},
		&stubResolver{windows: []model.ResolvedEvidenceWindow{{Candidate: model.VideoUnitCandidate{VideoID: "video-1", CoarseUnitID: 101}}}},
		&stubAggregator{videos: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		&stubRanker{ranked: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		&stubSelector{selected: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		&stubVideoFillService{},
		&stubExplainer{items: []model.FinalRecommendationItem{testFinalItem("video-1", 101, model.LearningRoleHardReview)}},
		&stubVideoStateEnricher{},
		writer,
	)
	if err != nil {
		t.Fatalf("NewGenerateVideoRecommendationsPipeline() error = %v", err)
	}

	response, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{
		UserID:           "user-1",
		TargetVideoCount: 3,
	})
	if err != nil {
		t.Fatalf("execute pipeline: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected one response item, got %#v", response.Items)
	}
	if writer.run.SelectorMode != "extreme_sparse" {
		t.Fatalf("expected extreme_sparse audit selector mode, got %q", writer.run.SelectorMode)
	}
	if !writer.run.Underfilled {
		t.Fatal("expected underfilled audit run")
	}
}

func TestGenerateVideoRecommendationsPipelineMapsLearningUnitEvidence(t *testing.T) {
	service, err := usecase.NewGenerateVideoRecommendationsPipeline(
		&constructorStubContextAssembler{
			context: model.RecommendationContext{
				PreferredDurationSec: [2]int{45, 200},
				Request:              model.RecommendationRequest{UserID: "user-1", TargetVideoCount: 1},
			},
		},
		&stubPlanner{bundle: model.DemandBundle{TargetVideoCount: 1}},
		&stubCandidateGenerator{candidates: []model.VideoUnitCandidate{{VideoID: "video-1", CoarseUnitID: 101}}},
		&stubResolver{windows: []model.ResolvedEvidenceWindow{{Candidate: model.VideoUnitCandidate{VideoID: "video-1", CoarseUnitID: 101}}}},
		&stubAggregator{videos: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		&stubRanker{ranked: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		&stubSelector{selected: []model.VideoCandidate{testVideoCandidate("video-1", 101, model.LearningRoleHardReview)}},
		&stubVideoFillService{},
		&stubExplainer{items: []model.FinalRecommendationItem{{
			VideoID:     "video-1",
			DurationMs:  90_000,
			Score:       0.91,
			ReasonCodes: []string{"hard_review_covered"},
			LearningUnits: []model.ExpectedLearningUnit{{
				CoarseUnitID: 101,
				Role:         model.LearningRoleHardReview,
				IsPrimary:    true,
				Evidence: &model.LearningUnitEvidence{
					SentenceIndex: int32Ptr(1),
					SpanIndex:     int32Ptr(2),
					StartMs:       int32Ptr(1240),
					EndMs:         int32Ptr(1820),
				},
			}},
		}}},
		&stubVideoStateEnricher{},
		&spyResultWriter{},
	)
	if err != nil {
		t.Fatalf("NewGenerateVideoRecommendationsPipeline() error = %v", err)
	}

	response, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{
		UserID:           "user-1",
		TargetVideoCount: 1,
	})
	if err != nil {
		t.Fatalf("execute pipeline: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 item, got %#v", response.Items)
	}
	if len(response.Items[0].LearningUnits) != 1 || response.Items[0].LearningUnits[0].Evidence == nil {
		t.Fatalf("expected learning unit evidence, got %#v", response.Items[0])
	}
	if response.Items[0].LearningUnits[0].Evidence.StartMs == nil || *response.Items[0].LearningUnits[0].Evidence.StartMs != 1240 {
		t.Fatalf("unexpected learning unit evidence bounds: %#v", response.Items[0].LearningUnits[0].Evidence)
	}
}

func TestGenerateVideoRecommendationsPipelinePersistsPrimaryLaneFromFullLaneSources(t *testing.T) {
	writer := &spyResultWriter{}
	selected := testVideoCandidate("video-1", 101, model.LearningRoleHardReview)
	selected.LaneSources = []string{"bundle", "exact_core"}

	service, err := usecase.NewGenerateVideoRecommendationsPipeline(
		&constructorStubContextAssembler{
			context: model.RecommendationContext{
				PreferredDurationSec: [2]int{45, 200},
				Request:              model.RecommendationRequest{UserID: "user-1", TargetVideoCount: 1},
			},
		},
		&stubPlanner{bundle: model.DemandBundle{TargetVideoCount: 1}},
		&stubCandidateGenerator{candidates: []model.VideoUnitCandidate{{VideoID: "video-1", CoarseUnitID: 101}}},
		&stubResolver{windows: []model.ResolvedEvidenceWindow{{Candidate: model.VideoUnitCandidate{VideoID: "video-1", CoarseUnitID: 101}}}},
		&stubAggregator{videos: []model.VideoCandidate{selected}},
		&stubRanker{ranked: []model.VideoCandidate{selected}},
		&stubSelector{selected: []model.VideoCandidate{selected}},
		&stubVideoFillService{},
		&stubExplainer{items: []model.FinalRecommendationItem{testFinalItem("video-1", 101, model.LearningRoleHardReview)}},
		&stubVideoStateEnricher{},
		writer,
	)
	if err != nil {
		t.Fatalf("NewGenerateVideoRecommendationsPipeline() error = %v", err)
	}

	if _, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{
		UserID:           "user-1",
		TargetVideoCount: 1,
	}); err != nil {
		t.Fatalf("execute pipeline: %v", err)
	}

	if len(writer.items) != 1 {
		t.Fatalf("expected one audit item, got %#v", writer.items)
	}
	if writer.items[0].PrimaryLane != "exact_core" {
		t.Fatalf("expected exact_core primary lane, got %#v", writer.items[0])
	}
	if writer.items[0].Rank != 1 {
		t.Fatalf("expected audit rank from selected item order, got %#v", writer.items[0])
	}
}

func TestGenerateVideoRecommendationsPipelineAuditsVideoFillItems(t *testing.T) {
	writer := &spyResultWriter{}
	learningVideo := testVideoCandidate("video-learning", 101, model.LearningRoleHardReview)
	fillVideo := model.VideoCandidate{
		VideoID:       "video-fill",
		DurationMs:    120_000,
		BaseScore:     0.51,
		LaneSources:   []string{string(policy.LaneMasteredTargetFill)},
		LearningUnits: []model.ExpectedLearningUnit{},
	}

	service, err := usecase.NewGenerateVideoRecommendationsPipeline(
		&constructorStubContextAssembler{
			context: model.RecommendationContext{
				PreferredDurationSec: [2]int{45, 200},
				Request:              model.RecommendationRequest{UserID: "user-1", TargetVideoCount: 2},
			},
		},
		&stubPlanner{bundle: model.DemandBundle{TargetVideoCount: 2}},
		&stubCandidateGenerator{candidates: []model.VideoUnitCandidate{{VideoID: "video-learning", CoarseUnitID: 101}}},
		&stubResolver{windows: []model.ResolvedEvidenceWindow{{Candidate: model.VideoUnitCandidate{VideoID: "video-learning", CoarseUnitID: 101}}}},
		&stubAggregator{videos: []model.VideoCandidate{learningVideo}},
		&stubRanker{ranked: []model.VideoCandidate{learningVideo}},
		&stubSelector{selected: []model.VideoCandidate{learningVideo}},
		&stubVideoFillService{filled: []model.VideoCandidate{learningVideo, fillVideo}},
		&mirrorSelectedExplainer{},
		&stubVideoStateEnricher{},
		writer,
	)
	if err != nil {
		t.Fatalf("NewGenerateVideoRecommendationsPipeline() error = %v", err)
	}

	response, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{
		UserID:           "user-1",
		TargetVideoCount: 2,
	})
	if err != nil {
		t.Fatalf("execute pipeline: %v", err)
	}

	if len(response.Items) != 2 {
		t.Fatalf("expected 2 response items, got %#v", response.Items)
	}
	if len(response.Items[1].LearningUnits) != 0 {
		t.Fatalf("fill response learning_units = %#v, want empty", response.Items[1].LearningUnits)
	}
	if len(writer.items) != 2 {
		t.Fatalf("expected 2 audit items, got %#v", writer.items)
	}
	if writer.items[1].PrimaryLane != string(policy.LaneMasteredTargetFill) {
		t.Fatalf("fill primary lane = %q", writer.items[1].PrimaryLane)
	}
	if writer.items[1].DominantUnitID != nil {
		t.Fatalf("fill dominant unit id = %#v, want nil", writer.items[1].DominantUnitID)
	}
	if len(writer.items[1].LearningUnits) != 0 {
		t.Fatalf("fill audit learning_units = %#v, want empty", writer.items[1].LearningUnits)
	}

	var summary map[string]any
	if err := json.Unmarshal(writer.run.CandidateSummary, &summary); err != nil {
		t.Fatalf("unmarshal candidate summary: %v", err)
	}
	if summary["fill_triggered"] != true {
		t.Fatalf("candidate summary fill_triggered = %#v", summary["fill_triggered"])
	}
	if summary["mastered_target_fill_count"] != float64(1) {
		t.Fatalf("candidate summary mastered fill count = %#v", summary["mastered_target_fill_count"])
	}
}

type stubPlanner struct{ bundle model.DemandBundle }

func (s *stubPlanner) Plan(model.RecommendationContext) (model.DemandBundle, error) {
	return s.bundle, nil
}

var _ domainplanner.DemandPlanner = (*stubPlanner)(nil)

type stubCandidateGenerator struct{ candidates []model.VideoUnitCandidate }

func (s *stubCandidateGenerator) Generate(context.Context, model.RecommendationContext, model.DemandBundle) ([]model.VideoUnitCandidate, error) {
	return s.candidates, nil
}

var _ domaincandidate.CandidateGenerator = (*stubCandidateGenerator)(nil)

type stubResolver struct {
	windows []model.ResolvedEvidenceWindow
}

func (s *stubResolver) Resolve(context.Context, model.RecommendationContext, []model.VideoUnitCandidate, model.DemandBundle) ([]model.ResolvedEvidenceWindow, error) {
	return s.windows, nil
}

var _ domainresolver.EvidenceResolver = (*stubResolver)(nil)

type stubAggregator struct{ videos []model.VideoCandidate }

func (s *stubAggregator) Aggregate(model.RecommendationContext, []model.ResolvedEvidenceWindow, model.DemandBundle) ([]model.VideoCandidate, error) {
	return s.videos, nil
}

var _ domainaggregator.VideoEvidenceAggregator = (*stubAggregator)(nil)

type stubRanker struct {
	ranked                 []model.VideoCandidate
	lastContextVideoStates int
	lastContextUserStates  int
}

func (s *stubRanker) Rank(contextModel model.RecommendationContext, _ []model.VideoCandidate, _ model.DemandBundle) ([]model.VideoCandidate, error) {
	s.lastContextVideoStates = len(contextModel.VideoServingStates)
	s.lastContextUserStates = len(contextModel.VideoUserStates)
	return s.ranked, nil
}

var _ domainranking.VideoRanker = (*stubRanker)(nil)

type stubSelector struct{ selected []model.VideoCandidate }

func (s *stubSelector) Select(model.RecommendationContext, []model.VideoCandidate, model.DemandBundle) ([]model.VideoCandidate, error) {
	return s.selected, nil
}

var _ domainselector.VideoSelector = (*stubSelector)(nil)

type stubVideoFillService struct {
	filled []model.VideoCandidate
}

func (s *stubVideoFillService) Fill(_ context.Context, _ model.RecommendationContext, selected []model.VideoCandidate, _ int) ([]model.VideoCandidate, error) {
	if s.filled != nil {
		return s.filled, nil
	}
	return selected, nil
}

var _ appservice.VideoFillService = (*stubVideoFillService)(nil)

type stubExplainer struct {
	items []model.FinalRecommendationItem
}

func (s *stubExplainer) Build(model.RecommendationContext, []model.VideoCandidate, model.DemandBundle) ([]model.FinalRecommendationItem, error) {
	return s.items, nil
}

var _ domainexplain.ExplanationBuilder = (*stubExplainer)(nil)

type mirrorSelectedExplainer struct{}

func (e *mirrorSelectedExplainer) Build(_ model.RecommendationContext, selected []model.VideoCandidate, _ model.DemandBundle) ([]model.FinalRecommendationItem, error) {
	items := make([]model.FinalRecommendationItem, 0, len(selected))
	for _, video := range selected {
		items = append(items, model.FinalRecommendationItem{
			VideoID:       video.VideoID,
			DurationMs:    video.DurationMs,
			Score:         video.BaseScore,
			LearningUnits: append([]model.ExpectedLearningUnit(nil), video.LearningUnits...),
		})
	}
	return items, nil
}

var _ domainexplain.ExplanationBuilder = (*mirrorSelectedExplainer)(nil)

type stubVideoStateEnricher struct {
	videoServingStates []model.UserVideoServingState
	videoUserStates    []model.VideoUserState
}

func (s *stubVideoStateEnricher) Enrich(_ context.Context, contextModel model.RecommendationContext, _ []model.VideoCandidate) (model.RecommendationContext, error) {
	contextModel.VideoServingStates = append([]model.UserVideoServingState(nil), s.videoServingStates...)
	contextModel.VideoUserStates = append([]model.VideoUserState(nil), s.videoUserStates...)
	return contextModel, nil
}

var _ appservice.VideoStateEnricher = (*stubVideoStateEnricher)(nil)

type spyResultWriter struct {
	called bool
	run    model.RecommendationRun
	items  []model.RecommendationItem
}

func (s *spyResultWriter) Persist(_ context.Context, run model.RecommendationRun, items []model.RecommendationItem, _ string, _ []model.FinalRecommendationItem) error {
	s.called = true
	s.run = run
	s.items = append([]model.RecommendationItem(nil), items...)
	return nil
}

var _ appservice.RecommendationResultWriter = (*spyResultWriter)(nil)

func testVideoCandidate(videoID string, unitID int64, role model.LearningRole) model.VideoCandidate {
	return model.VideoCandidate{
		VideoID:        videoID,
		DurationMs:     90_000,
		BaseScore:      0.91,
		DominantRole:   role,
		DominantUnitID: int64Ptr(unitID),
		LaneSources:    []string{"exact_core"},
		LearningUnits: []model.ExpectedLearningUnit{{
			CoarseUnitID: unitID,
			Role:         role,
			IsPrimary:    true,
		}},
	}
}

func testFinalItem(videoID string, unitID int64, role model.LearningRole) model.FinalRecommendationItem {
	return model.FinalRecommendationItem{
		VideoID:     videoID,
		DurationMs:  90_000,
		Score:       0.91,
		ReasonCodes: []string{"hard_review_covered"},
		LearningUnits: []model.ExpectedLearningUnit{{
			CoarseUnitID: unitID,
			Role:         role,
			IsPrimary:    true,
		}},
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}
