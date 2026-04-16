package service

import (
	"context"
	"math"
	"sort"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	domaincandidate "learning-video-recommendation-system/internal/recommendation/domain/candidate"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

type DefaultCandidateGenerator struct {
	recommendable apprepo.RecommendableVideoUnitReader
}

var _ domaincandidate.CandidateGenerator = (*DefaultCandidateGenerator)(nil)

func NewDefaultCandidateGenerator(recommendable apprepo.RecommendableVideoUnitReader) *DefaultCandidateGenerator {
	return &DefaultCandidateGenerator{
		recommendable: recommendable,
	}
}

func (g *DefaultCandidateGenerator) Generate(ctx context.Context, recommendationContext model.RecommendationContext, demand model.DemandBundle) ([]model.VideoUnitCandidate, error) {
	rows, err := g.loadRecommendableRows(ctx, recommendationContext, demand)
	if err != nil {
		return nil, err
	}

	exactCandidates := g.generateExactCore(rows, demand)
	bundleCandidates := g.generateBundle(rows, demand)
	softFutureCandidates := g.generateSoftFuture(rows, demand)

	existingVideos := make(map[string]struct{})
	for _, candidate := range append(append(exactCandidates, bundleCandidates...), softFutureCandidates...) {
		existingVideos[candidate.VideoID] = struct{}{}
	}
	fallbackCandidates := g.generateQualityFallback(rows, demand, existingVideos)

	candidates := make([]model.VideoUnitCandidate, 0, len(exactCandidates)+len(bundleCandidates)+len(softFutureCandidates)+len(fallbackCandidates))
	candidates = append(candidates, exactCandidates...)
	candidates = append(candidates, bundleCandidates...)
	candidates = append(candidates, softFutureCandidates...)
	candidates = append(candidates, fallbackCandidates...)

	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if lanePriority(left.Lane) != lanePriority(right.Lane) {
			return lanePriority(left.Lane) < lanePriority(right.Lane)
		}
		if left.CandidateScore != right.CandidateScore {
			return left.CandidateScore > right.CandidateScore
		}
		if left.VideoID != right.VideoID {
			return left.VideoID < right.VideoID
		}
		return left.CoarseUnitID < right.CoarseUnitID
	})

	return candidates, nil
}

func (g *DefaultCandidateGenerator) loadRecommendableRows(ctx context.Context, recommendationContext model.RecommendationContext, demand model.DemandBundle) ([]model.RecommendableVideoUnit, error) {
	if len(recommendationContext.RecommendableVideoUnit) > 0 {
		return recommendationContext.RecommendableVideoUnit, nil
	}

	unitIDs := demandUnitIDs(demand)
	if len(unitIDs) == 0 {
		return []model.RecommendableVideoUnit{}, nil
	}

	return g.recommendable.ListByUnitIDs(ctx, unitIDs)
}

func (g *DefaultCandidateGenerator) generateExactCore(rows []model.RecommendableVideoUnit, demand model.DemandBundle) []model.VideoUnitCandidate {
	demandByUnit := demandUnitsByID(demand)
	coreUnitIDs := make(map[int64]struct{})
	for _, unit := range demand.HardReview {
		coreUnitIDs[unit.UnitID] = struct{}{}
	}
	for _, unit := range topDemandUnits(demand.NewNow, maxInt(1, int(math.Ceil(float64(demand.TargetVideoCount)*demand.LaneBudget.ExactCore*0.5)))) {
		coreUnitIDs[unit.UnitID] = struct{}{}
	}

	filtered := make([]model.VideoUnitCandidate, 0)
	for _, row := range rows {
		demandUnit, ok := demandByUnit[row.CoarseUnitID]
		if !ok {
			continue
		}
		if _, ok := coreUnitIDs[row.CoarseUnitID]; !ok {
			continue
		}
		filtered = append(filtered, videoUnitCandidateFromRow(row, demandUnit, string(policy.LaneExactCore), exactCoreScore(row, demandUnit)))
	}

	return capCandidatesByDistinctVideos(filtered, maxInt(2, int(math.Ceil(float64(demand.TargetVideoCount)*demand.LaneBudget.ExactCore))))
}

func (g *DefaultCandidateGenerator) generateBundle(rows []model.RecommendableVideoUnit, demand model.DemandBundle) []model.VideoUnitCandidate {
	demandByUnit := demandUnitsByID(demand)
	inputUnitIDs := make(map[int64]struct{})
	for _, unit := range demand.HardReview {
		inputUnitIDs[unit.UnitID] = struct{}{}
	}
	for _, unit := range demand.NewNow {
		inputUnitIDs[unit.UnitID] = struct{}{}
	}
	for _, unit := range demand.SoftReview {
		inputUnitIDs[unit.UnitID] = struct{}{}
	}
	for _, unit := range topDemandUnits(demand.NearFuture, maxInt(1, int(math.Ceil(float64(demand.TargetVideoCount)*0.25)))) {
		inputUnitIDs[unit.UnitID] = struct{}{}
	}

	rowsByVideo := make(map[string]map[int64]bundleRow)
	for _, row := range rows {
		if _, ok := inputUnitIDs[row.CoarseUnitID]; !ok {
			continue
		}
		demandUnit, ok := demandByUnit[row.CoarseUnitID]
		if !ok {
			continue
		}
		if _, ok := rowsByVideo[row.VideoID]; !ok {
			rowsByVideo[row.VideoID] = make(map[int64]bundleRow)
		}
		scored := bundleRow{row: row, demand: demandUnit, score: bundleUnitScore(row, demandUnit)}
		existing, ok := rowsByVideo[row.VideoID][row.CoarseUnitID]
		if !ok || scored.score > existing.score {
			rowsByVideo[row.VideoID][row.CoarseUnitID] = scored
		}
	}

	type bundleVideo struct {
		videoID string
		rows    []bundleRow
		score   float64
	}
	qualified := make([]bundleVideo, 0)
	for videoID, unitRows := range rowsByVideo {
		selected := make([]bundleRow, 0, len(unitRows))
		hasCore := false
		softCount := 0
		for _, item := range unitRows {
			selected = append(selected, item)
			if item.demand.Bucket == string(policy.BucketHardReview) || item.demand.Bucket == string(policy.BucketNewNow) {
				hasCore = true
			}
			if item.demand.Bucket == string(policy.BucketSoftReview) {
				softCount++
			}
		}
		if len(selected) < 2 {
			continue
		}
		if !hasCore && !(demand.Flags.HardReviewLowSupply && softCount >= 2) {
			continue
		}

		sort.SliceStable(selected, func(i, j int) bool {
			if bucketPriority(selected[i].demand.Bucket) != bucketPriority(selected[j].demand.Bucket) {
				return bucketPriority(selected[i].demand.Bucket) < bucketPriority(selected[j].demand.Bucket)
			}
			if selected[i].score != selected[j].score {
				return selected[i].score > selected[j].score
			}
			return selected[i].demand.UnitID < selected[j].demand.UnitID
		})

		qualified = append(qualified, bundleVideo{
			videoID: videoID,
			rows:    selected,
			score:   bundleVideoScore(selected, hasCore),
		})
	}

	sort.SliceStable(qualified, func(i, j int) bool {
		if qualified[i].score != qualified[j].score {
			return qualified[i].score > qualified[j].score
		}
		return qualified[i].videoID < qualified[j].videoID
	})

	capDistinctVideos := maxInt(2, int(math.Ceil(float64(demand.TargetVideoCount)*demand.LaneBudget.Bundle)))
	candidates := make([]model.VideoUnitCandidate, 0)
	for _, video := range qualified {
		if capDistinctVideos <= 0 {
			break
		}
		capDistinctVideos--
		for _, item := range video.rows {
			candidate := videoUnitCandidateFromRow(item.row, item.demand, string(policy.LaneBundle), round4(item.score+bundleVideoLaneBonus(len(video.rows), item.demand.Bucket)))
			candidates = append(candidates, candidate)
		}
	}

	return candidates
}

func (g *DefaultCandidateGenerator) generateSoftFuture(rows []model.RecommendableVideoUnit, demand model.DemandBundle) []model.VideoUnitCandidate {
	demandByUnit := demandUnitsByID(demand)
	eligible := make(map[int64]struct{})
	for _, unit := range demand.SoftReview {
		eligible[unit.UnitID] = struct{}{}
	}
	for _, unit := range demand.NearFuture {
		eligible[unit.UnitID] = struct{}{}
	}

	filtered := make([]model.VideoUnitCandidate, 0)
	for _, row := range rows {
		if _, ok := eligible[row.CoarseUnitID]; !ok {
			continue
		}
		demandUnit, ok := demandByUnit[row.CoarseUnitID]
		if !ok {
			continue
		}
		filtered = append(filtered, videoUnitCandidateFromRow(row, demandUnit, string(policy.LaneSoftFuture), softFutureScore(row, demandUnit)))
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if bucketPriority(filtered[i].Bucket) != bucketPriority(filtered[j].Bucket) {
			return bucketPriority(filtered[i].Bucket) < bucketPriority(filtered[j].Bucket)
		}
		if filtered[i].CandidateScore != filtered[j].CandidateScore {
			return filtered[i].CandidateScore > filtered[j].CandidateScore
		}
		if filtered[i].VideoID != filtered[j].VideoID {
			return filtered[i].VideoID < filtered[j].VideoID
		}
		return filtered[i].CoarseUnitID < filtered[j].CoarseUnitID
	})

	return capCandidatesByDistinctVideos(filtered, maxInt(2, int(math.Ceil(float64(demand.TargetVideoCount)*(demand.LaneBudget.SoftFuture+0.25)))))
}

func (g *DefaultCandidateGenerator) generateQualityFallback(rows []model.RecommendableVideoUnit, demand model.DemandBundle, existingVideos map[string]struct{}) []model.VideoUnitCandidate {
	if len(existingVideos) >= demand.TargetVideoCount {
		return nil
	}

	demandByUnit := demandUnitsByID(demand)
	filtered := make([]model.VideoUnitCandidate, 0)
	for _, row := range rows {
		if _, exists := existingVideos[row.VideoID]; exists {
			continue
		}
		demandUnit, ok := demandByUnit[row.CoarseUnitID]
		if !ok {
			continue
		}
		filtered = append(filtered, videoUnitCandidateFromRow(row, demandUnit, string(policy.LaneQualityFallback), fallbackScore(row, demandUnit, demand.PreferredDurationSec)))
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].CandidateScore != filtered[j].CandidateScore {
			return filtered[i].CandidateScore > filtered[j].CandidateScore
		}
		if filtered[i].VideoID != filtered[j].VideoID {
			return filtered[i].VideoID < filtered[j].VideoID
		}
		return filtered[i].CoarseUnitID < filtered[j].CoarseUnitID
	})

	gap := demand.TargetVideoCount - len(existingVideos)
	capDistinctVideos := minInt(1, gap)
	if capDistinctVideos <= 0 {
		return nil
	}
	return capCandidatesByDistinctVideos(filtered, capDistinctVideos)
}

func videoUnitCandidateFromRow(row model.RecommendableVideoUnit, demandUnit model.DemandUnit, lane string, candidateScore float64) model.VideoUnitCandidate {
	return model.VideoUnitCandidate{
		VideoID:            row.VideoID,
		CoarseUnitID:       row.CoarseUnitID,
		Lane:               lane,
		Bucket:             demandUnit.Bucket,
		UnitWeight:         demandUnit.Weight,
		MentionCount:       row.MentionCount,
		SentenceCount:      row.SentenceCount,
		CoverageMs:         row.CoverageMs,
		CoverageRatio:      row.CoverageRatio,
		SentenceIndexes:    row.SentenceIndexes,
		EvidenceSpanRefs:   row.EvidenceSpanRefs,
		SampleSurfaceForms: row.SampleSurfaceForms,
		DurationMs:         row.DurationMs,
		MappedSpanRatio:    row.MappedSpanRatio,
		CandidateScore:     candidateScore,
	}
}

func exactCoreScore(row model.RecommendableVideoUnit, demandUnit model.DemandUnit) float64 {
	return round4(demandUnit.Weight*0.55 + coverageStrength(row)*0.45)
}

func bundleUnitScore(row model.RecommendableVideoUnit, demandUnit model.DemandUnit) float64 {
	return round4(demandUnit.Weight*0.50 + coverageStrength(row)*0.35 + bundleBucketBonus(demandUnit.Bucket))
}

func softFutureScore(row model.RecommendableVideoUnit, demandUnit model.DemandUnit) float64 {
	return round4(demandUnit.Weight*0.45 + coverageStrength(row)*0.35 + bundleBucketBonus(demandUnit.Bucket))
}

func fallbackScore(row model.RecommendableVideoUnit, demandUnit model.DemandUnit, preferredDurationSec [2]int) float64 {
	return round4(durationFit(row.DurationMs, preferredDurationSec)*0.40 + coverageStrength(row)*0.35 + demandUnit.Weight*0.25)
}

func coverageStrength(row model.RecommendableVideoUnit) float64 {
	mentionScore := math.Min(float64(row.MentionCount)/4.0, 1.0)
	sentenceScore := math.Min(float64(row.SentenceCount)/3.0, 1.0)
	return round4(
		row.CoverageRatio*0.40 +
			mentionScore*0.20 +
			sentenceScore*0.15 +
			row.MappedSpanRatio*0.25,
	)
}

func bundleVideoScore(rows []bundleRow, hasCore bool) float64 {
	total := 0.0
	for _, row := range rows {
		total += row.score
	}

	coreBonus := 0.0
	if hasCore {
		coreBonus = 0.20
	}

	return round4(total/float64(len(rows)) + float64(len(rows))*0.10 + coreBonus)
}

type bundleRow struct {
	row    model.RecommendableVideoUnit
	demand model.DemandUnit
	score  float64
}

func bundleVideoLaneBonus(bundleSize int, bucket string) float64 {
	return round4(float64(bundleSize)*0.05 + bundleBucketBonus(bucket))
}

func bundleBucketBonus(bucket string) float64 {
	switch bucket {
	case string(policy.BucketHardReview):
		return 0.20
	case string(policy.BucketNewNow):
		return 0.15
	case string(policy.BucketSoftReview):
		return 0.10
	default:
		return 0.05
	}
}

func durationFit(durationMs int32, preferredDurationSec [2]int) float64 {
	minMs := preferredDurationSec[0] * 1000
	maxMs := preferredDurationSec[1] * 1000
	if int(durationMs) >= minMs && int(durationMs) <= maxMs {
		return 1.0
	}
	if int(durationMs) < minMs {
		delta := float64(minMs-int(durationMs)) / float64(minMs)
		return round4(math.Max(0.0, 1.0-delta))
	}
	delta := float64(int(durationMs)-maxMs) / float64(maxMs)
	return round4(math.Max(0.0, 1.0-delta))
}

func capCandidatesByDistinctVideos(candidates []model.VideoUnitCandidate, distinctVideoCap int) []model.VideoUnitCandidate {
	if distinctVideoCap <= 0 {
		return nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].CandidateScore != candidates[j].CandidateScore {
			return candidates[i].CandidateScore > candidates[j].CandidateScore
		}
		if bucketPriority(candidates[i].Bucket) != bucketPriority(candidates[j].Bucket) {
			return bucketPriority(candidates[i].Bucket) < bucketPriority(candidates[j].Bucket)
		}
		if candidates[i].VideoID != candidates[j].VideoID {
			return candidates[i].VideoID < candidates[j].VideoID
		}
		return candidates[i].CoarseUnitID < candidates[j].CoarseUnitID
	})

	selected := make([]model.VideoUnitCandidate, 0, len(candidates))
	allowedVideos := make(map[string]struct{})
	for _, candidate := range candidates {
		if _, exists := allowedVideos[candidate.VideoID]; exists {
			selected = append(selected, candidate)
			continue
		}
		if len(allowedVideos) >= distinctVideoCap {
			continue
		}
		allowedVideos[candidate.VideoID] = struct{}{}
		selected = append(selected, candidate)
	}
	return selected
}

func demandUnitIDs(demand model.DemandBundle) []int64 {
	seen := make(map[int64]struct{})
	result := make([]int64, 0, len(demand.HardReview)+len(demand.NewNow)+len(demand.SoftReview)+len(demand.NearFuture))
	for _, units := range [][]model.DemandUnit{demand.HardReview, demand.NewNow, demand.SoftReview, demand.NearFuture} {
		for _, unit := range units {
			if _, ok := seen[unit.UnitID]; ok {
				continue
			}
			seen[unit.UnitID] = struct{}{}
			result = append(result, unit.UnitID)
		}
	}
	return result
}

func demandUnitsByID(demand model.DemandBundle) map[int64]model.DemandUnit {
	result := make(map[int64]model.DemandUnit, len(demand.HardReview)+len(demand.NewNow)+len(demand.SoftReview)+len(demand.NearFuture))
	for _, units := range [][]model.DemandUnit{demand.HardReview, demand.NewNow, demand.SoftReview, demand.NearFuture} {
		for _, unit := range units {
			result[unit.UnitID] = unit
		}
	}
	return result
}

func topDemandUnits(units []model.DemandUnit, cap int) []model.DemandUnit {
	if cap <= 0 || len(units) == 0 {
		return nil
	}

	cloned := append([]model.DemandUnit(nil), units...)
	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].Weight != cloned[j].Weight {
			return cloned[i].Weight > cloned[j].Weight
		}
		return cloned[i].UnitID < cloned[j].UnitID
	})
	if cap > len(cloned) {
		cap = len(cloned)
	}
	return cloned[:cap]
}

func lanePriority(lane string) int {
	switch lane {
	case string(policy.LaneExactCore):
		return 0
	case string(policy.LaneBundle):
		return 1
	case string(policy.LaneSoftFuture):
		return 2
	default:
		return 3
	}
}

func bucketPriority(bucket string) int {
	switch bucket {
	case string(policy.BucketHardReview):
		return 0
	case string(policy.BucketNewNow):
		return 1
	case string(policy.BucketSoftReview):
		return 2
	default:
		return 3
	}
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
