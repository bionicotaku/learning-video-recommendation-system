package explain_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	recommendationexplain "learning-video-recommendation-system/internal/recommendation/domain/explain"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

func TestDefaultExplanationBuilderGeneratesReasonCodesAndNarrative(t *testing.T) {
	builder := recommendationexplain.NewDefaultExplanationBuilder()
	start := int32(1240)
	end := int32(1820)

	items, err := builder.Build(model.RecommendationContext{}, []model.VideoCandidate{
		{
			VideoID:                "video-1",
			BaseScore:              0.91,
			LaneSources:            []string{string(policy.LaneBundle)},
			DominantBucket:         string(policy.BucketHardReview),
			DominantUnitID:         int64Ptr(101),
			CoveredHardReviewUnits: []int64{101, 102},
			CoveredNewNowUnits:     []int64{201},
			CoveredSoftReviewUnits: []int64{301},
			BundleValueScore:       0.7,
			EducationalFitScore:    0.8,
			RecentServedPenalty:    0.1,
			BestEvidenceStartMs:    &start,
			BestEvidenceEndMs:      &end,
		},
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("build explanation: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected one final item, got %#v", items)
	}
	if !contains(items[0].ReasonCodes, string(policy.ReasonCodeHardReviewCovered)) {
		t.Fatalf("expected hard_review_covered reason code, got %#v", items[0].ReasonCodes)
	}
	if !contains(items[0].ReasonCodes, string(policy.ReasonCodeBundleCoverageHigh)) {
		t.Fatalf("expected bundle_coverage_high reason code, got %#v", items[0].ReasonCodes)
	}
	if items[0].Explanation == "" {
		t.Fatal("expected non-empty explanation")
	}
}

func TestDefaultExplanationBuilderGoldenFinalOrdering(t *testing.T) {
	builder := recommendationexplain.NewDefaultExplanationBuilder()
	start := int32(1240)
	end := int32(1820)
	start2 := int32(3000)
	end2 := int32(3560)

	items, err := builder.Build(model.RecommendationContext{}, []model.VideoCandidate{
		{
			VideoID:                   "video-hard",
			BaseScore:                 0.93,
			LaneSources:               []string{string(policy.LaneExactCore), string(policy.LaneBundle)},
			DominantBucket:            string(policy.BucketHardReview),
			DominantUnitID:            int64Ptr(101),
			CoveredHardReviewUnits:    []int64{101},
			CoveredSoftReviewUnits:    []int64{301},
			BundleValueScore:          0.6,
			EducationalFitScore:       0.8,
			BestEvidenceStartMs:       &start,
			BestEvidenceEndMs:         &end,
			BestEvidenceSentenceIndex: int32Ptr(1),
			BestEvidenceSpanIndex:     int32Ptr(1),
		},
		{
			VideoID:                   "video-future",
			BaseScore:                 0.74,
			LaneSources:               []string{string(policy.LaneSoftFuture)},
			DominantBucket:            string(policy.BucketNearFuture),
			DominantUnitID:            int64Ptr(401),
			CoveredNearFutureUnits:    []int64{401},
			EducationalFitScore:       0.7,
			BestEvidenceStartMs:       &start2,
			BestEvidenceEndMs:         &end2,
			BestEvidenceSentenceIndex: int32Ptr(3),
			BestEvidenceSpanIndex:     int32Ptr(1),
		},
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("build explanation: %v", err)
	}

	actual, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		t.Fatalf("marshal items: %v", err)
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	goldenPath := filepath.Join(filepath.Dir(currentFile), "../../../golden/final_ordering_normal.json")
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
		t.Fatalf("final ordering golden mismatch\nactual:\n%s\nexpected:\n%s", actual, expected)
	}
}

func recommendationDemand() model.DemandBundle {
	return model.DemandBundle{
		Flags: model.PlannerFlags{HardReviewLowSupply: true},
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func int64Ptr(value int64) *int64 {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}
