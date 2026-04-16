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
	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/application/usecase"
	domainaggregator "learning-video-recommendation-system/internal/recommendation/domain/aggregator"
	domaincandidate "learning-video-recommendation-system/internal/recommendation/domain/candidate"
	domainexplain "learning-video-recommendation-system/internal/recommendation/domain/explain"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	domainplanner "learning-video-recommendation-system/internal/recommendation/domain/planner"
	domainranking "learning-video-recommendation-system/internal/recommendation/domain/ranking"
	domainresolver "learning-video-recommendation-system/internal/recommendation/domain/resolver"
	domainselector "learning-video-recommendation-system/internal/recommendation/domain/selector"
)

func TestGenerateVideoRecommendationsPipelineExecutesFullRecommendationFlow(t *testing.T) {
	writer := &spyResultWriter{}
	ranker := &stubRanker{
		ranked: []model.VideoCandidate{
			{
				VideoID:                "video-1",
				BaseScore:              0.91,
				DominantBucket:         "hard_review",
				DominantUnitID:         int64Ptr(101),
				LaneSources:            []string{"exact_core"},
				CoveredHardReviewUnits: []int64{101},
			},
		},
	}

	service := usecase.NewGenerateVideoRecommendationsPipeline(
		&stubContextAssembler{
			context: model.RecommendationContext{
				Request: model.RecommendationRequest{UserID: "user-1", TargetVideoCount: 2, PreferredDurationSec: [2]int{45, 180}},
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
		&stubAggregator{videos: []model.VideoCandidate{{VideoID: "video-1", DominantBucket: "hard_review", DominantUnitID: int64Ptr(101), LaneSources: []string{"exact_core"}, CoveredHardReviewUnits: []int64{101}}}},
		ranker,
		&stubSelector{selected: ranker.ranked},
		&stubExplainer{items: []model.FinalRecommendationItem{{VideoID: "video-1", Rank: 1, Score: 0.91, ReasonCodes: []string{"hard_review_covered"}, CoveredHardReviewUnits: []int64{101}, Explanation: "ok"}}},
		&stubVideoServingStateRepo{states: []model.UserVideoServingState{{VideoID: "video-1"}}},
		&stubVideoUserStateReader{states: []model.VideoUserState{{VideoID: "video-1", WatchCount: 1}}},
		writer,
	)

	response, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{
		UserID:               "user-1",
		TargetVideoCount:     2,
		PreferredDurationSec: [2]int{45, 180},
	})
	if err != nil {
		t.Fatalf("execute pipeline: %v", err)
	}

	if response.RunID == "" {
		t.Fatal("expected generated run id")
	}
	if response.SelectorMode != "low_supply" {
		t.Fatalf("expected low_supply selector mode, got %q", response.SelectorMode)
	}
	if len(response.Videos) != 1 || response.Videos[0].VideoID != "video-1" {
		t.Fatalf("unexpected response videos: %#v", response.Videos)
	}
	if !writer.called {
		t.Fatal("expected result writer to persist outputs")
	}
	if ranker.lastContextVideoStates != 1 {
		t.Fatalf("expected ranker to receive loaded video serving states, got %d", ranker.lastContextVideoStates)
	}
	if ranker.lastContextUserStates != 1 {
		t.Fatalf("expected ranker to receive loaded video user states, got %d", ranker.lastContextUserStates)
	}
}

func TestGenerateVideoRecommendationsPipelineGoldenResponse(t *testing.T) {
	service := usecase.NewGenerateVideoRecommendationsPipeline(
		&stubContextAssembler{
			context: model.RecommendationContext{
				Request: model.RecommendationRequest{UserID: "user-1", TargetVideoCount: 2, PreferredDurationSec: [2]int{45, 180}},
			},
		},
		&stubPlanner{bundle: model.DemandBundle{TargetVideoCount: 2}},
		&stubCandidateGenerator{candidates: []model.VideoUnitCandidate{{VideoID: "video-1", CoarseUnitID: 101}}},
		&stubResolver{windows: []model.ResolvedEvidenceWindow{{Candidate: model.VideoUnitCandidate{VideoID: "video-1", CoarseUnitID: 101}}}},
		&stubAggregator{videos: []model.VideoCandidate{{VideoID: "video-1", DominantBucket: "hard_review", DominantUnitID: int64Ptr(101), LaneSources: []string{"exact_core"}, CoveredHardReviewUnits: []int64{101}}}},
		&stubRanker{ranked: []model.VideoCandidate{{VideoID: "video-1", BaseScore: 0.91, DominantBucket: "hard_review", DominantUnitID: int64Ptr(101), LaneSources: []string{"exact_core"}, CoveredHardReviewUnits: []int64{101}}}},
		&stubSelector{selected: []model.VideoCandidate{{VideoID: "video-1", BaseScore: 0.91, DominantBucket: "hard_review", DominantUnitID: int64Ptr(101), LaneSources: []string{"exact_core"}, CoveredHardReviewUnits: []int64{101}}}},
		&stubExplainer{items: []model.FinalRecommendationItem{{VideoID: "video-1", Rank: 1, Score: 0.91, ReasonCodes: []string{"hard_review_covered"}, CoveredHardReviewUnits: []int64{101}, Explanation: "ok"}}},
		&stubVideoServingStateRepo{},
		&stubVideoUserStateReader{},
		&spyResultWriter{},
	)

	response, err := service.Execute(context.Background(), dto.GenerateVideoRecommendationsRequest{
		UserID:               "user-1",
		TargetVideoCount:     2,
		PreferredDurationSec: [2]int{45, 180},
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

type stubPlanner struct{ bundle model.DemandBundle }

func (s *stubPlanner) Plan(model.RecommendationContext) (model.DemandBundle, error) {
	return s.bundle, nil
}

var _ domainplanner.DemandPlanner = (*stubPlanner)(nil)

type stubCandidateGenerator struct{ candidates []model.VideoUnitCandidate }

func (s *stubCandidateGenerator) Generate(model.RecommendationContext, model.DemandBundle) ([]model.VideoUnitCandidate, error) {
	return s.candidates, nil
}

var _ domaincandidate.CandidateGenerator = (*stubCandidateGenerator)(nil)

type stubResolver struct {
	windows []model.ResolvedEvidenceWindow
}

func (s *stubResolver) Resolve(model.RecommendationContext, []model.VideoUnitCandidate, model.DemandBundle) ([]model.ResolvedEvidenceWindow, error) {
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

type stubExplainer struct {
	items []model.FinalRecommendationItem
}

func (s *stubExplainer) Build(model.RecommendationContext, []model.VideoCandidate, model.DemandBundle) ([]model.FinalRecommendationItem, error) {
	return s.items, nil
}

var _ domainexplain.ExplanationBuilder = (*stubExplainer)(nil)

type stubVideoServingStateRepo struct{ states []model.UserVideoServingState }

func (s *stubVideoServingStateRepo) ListByUserAndVideoIDs(context.Context, string, []string) ([]model.UserVideoServingState, error) {
	return s.states, nil
}
func (s *stubVideoServingStateRepo) Upsert(context.Context, model.UserVideoServingState) error {
	return nil
}

var _ apprepo.VideoServingStateRepository = (*stubVideoServingStateRepo)(nil)

type stubVideoUserStateReader struct{ states []model.VideoUserState }

func (s *stubVideoUserStateReader) ListByUserAndVideoIDs(context.Context, string, []string) ([]model.VideoUserState, error) {
	return s.states, nil
}

var _ apprepo.VideoUserStateReader = (*stubVideoUserStateReader)(nil)

type spyResultWriter struct {
	called bool
}

func (s *spyResultWriter) Persist(context.Context, model.RecommendationRun, []model.RecommendationItem, string, []model.FinalRecommendationItem) error {
	s.called = true
	return nil
}

var _ appservice.RecommendationResultWriter = (*spyResultWriter)(nil)

func int64Ptr(value int64) *int64 {
	return &value
}
