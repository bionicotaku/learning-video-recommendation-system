package explain

import (
	"fmt"
	"strings"

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
	for index, video := range selected {
		reasonCodes := buildReasonCodes(video, demand)
		items = append(items, model.FinalRecommendationItem{
			VideoID:       video.VideoID,
			Rank:          index + 1,
			Score:         video.BaseScore,
			ReasonCodes:   reasonCodes,
			LearningUnits: append([]model.ExpectedLearningUnit(nil), video.LearningUnits...),
			Explanation:   explanationText(video),
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

func explanationText(video model.VideoCandidate) string {
	parts := make([]string, 0, 4)
	if count := model.CountLearningUnitsByRole(video.LearningUnits, model.LearningRoleHardReview); count > 0 {
		parts = append(parts, fmt.Sprintf("覆盖 %d 个当前应复习内容", count))
	}
	if count := model.CountLearningUnitsByRole(video.LearningUnits, model.LearningRoleNewNow); count > 0 {
		parts = append(parts, fmt.Sprintf("顺带覆盖 %d 个当前可引入的新内容", count))
	}
	if count := model.CountLearningUnitsByRole(video.LearningUnits, model.LearningRoleSoftReview); count > 0 {
		parts = append(parts, fmt.Sprintf("支持 %d 个近期不稳内容", count))
	}
	if evidence := model.PrimaryLearningUnitEvidence(video.LearningUnits); evidence != nil && evidence.StartMs != nil && evidence.EndMs != nil {
		parts = append(parts, fmt.Sprintf("主要学习证据集中在 %s–%s", formatMs(*evidence.StartMs), formatMs(*evidence.EndMs)))
	}
	if video.RecentServedPenalty < 0.20 {
		parts = append(parts, "最近未重复推荐")
	}
	return strings.Join(parts, "，")
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

func formatMs(value int32) string {
	totalSeconds := value / 1000
	centiseconds := (value % 1000) / 10
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, centiseconds)
}
