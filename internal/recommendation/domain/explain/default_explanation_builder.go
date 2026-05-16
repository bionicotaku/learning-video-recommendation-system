package explain

import (
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

type DefaultExplanationBuilder struct{}

var _ ExplanationBuilder = (*DefaultExplanationBuilder)(nil)

func NewDefaultExplanationBuilder() *DefaultExplanationBuilder {
	return &DefaultExplanationBuilder{}
}

func (b *DefaultExplanationBuilder) Build(recommendationContext model.RecommendationContext, selected []model.VideoCandidate, demand model.DemandBundle) ([]model.FinalRecommendationItem, error) {
	_ = recommendationContext

	items := make([]model.FinalRecommendationItem, 0, len(selected))
	for _, video := range selected {
		reasonCodes := buildReasonCodes(video, demand)
		items = append(items, model.FinalRecommendationItem{
			VideoID:       video.VideoID,
			DurationMs:    video.DurationMs,
			Score:         video.BaseScore,
			ReasonCodes:   reasonCodes,
			LearningUnits: append([]model.ExpectedLearningUnit(nil), video.LearningUnits...),
		})
	}
	return items, nil
}

func buildReasonCodes(video model.VideoCandidate, demand model.DemandBundle) []string {
	reasonCodes := make([]string, 0, 8)
	if model.HasLearningRole(video.LearningUnits, model.LearningRoleHardReview) {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeHardReviewCovered))
	}
	if model.HasLearningRole(video.LearningUnits, model.LearningRoleNewNow) {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeNewUnitIntroduced))
	}
	if model.HasLearningRole(video.LearningUnits, model.LearningRoleSoftReview) {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeSoftReviewSupport))
	}
	if model.HasLearningRole(video.LearningUnits, model.LearningRoleNearFuture) {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeNearFutureWarmup))
	}
	if len(model.LearningUnitIDs(video.LearningUnits)) >= 2 || video.BundleValueScore >= 0.50 {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeBundleCoverageHigh))
	}
	if model.PrimaryLearningUnitEvidence(video.LearningUnits) != nil {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeStrongEvidence))
	}
	if video.EducationalFitScore >= 0.65 {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeGoodLearningFit))
	}
	if video.RecentServedPenalty < 0.20 {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeRecentlyNotServed))
	}
	if demand.Flags.HardReviewLowSupply && model.HasLearningRole(video.LearningUnits, model.LearningRoleHardReview) {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeLowSupplyPreserve))
	}
	if len(video.LaneSources) == 1 && video.LaneSources[0] == string(policy.LaneQualityFallback) {
		reasonCodes = append(reasonCodes, string(policy.ReasonCodeFallbackQuality))
	}
	return uniqueStrings(reasonCodes)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
