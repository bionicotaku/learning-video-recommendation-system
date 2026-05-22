package service

import (
	"context"
	"math"
	"sort"
	"time"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

const maxVideoFillQueryLimit int32 = 30

type DefaultVideoFillService struct {
	reader apprepo.VideoFillCandidateReader
	now    func() time.Time
}

var _ VideoFillService = (*DefaultVideoFillService)(nil)

func NewDefaultVideoFillService(reader apprepo.VideoFillCandidateReader) *DefaultVideoFillService {
	return &DefaultVideoFillService{
		reader: reader,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *DefaultVideoFillService) Fill(ctx context.Context, contextModel model.RecommendationContext, selected []model.VideoCandidate, targetCount int) ([]model.VideoCandidate, error) {
	if targetCount <= 0 || len(selected) >= targetCount || s.reader == nil {
		return selected, nil
	}

	filled := append([]model.VideoCandidate(nil), selected...)
	excluded := videoFillExcludedSet(filled)

	gap := targetCount - len(filled)
	mastered, err := s.reader.ListMasteredTargetFillCandidates(ctx, contextModel.Request.UserID, videoFillExcludedList(excluded), videoFillQueryLimit(gap))
	if err != nil {
		return nil, err
	}
	filled = appendFillCandidates(filled, mastered, string(policy.LaneMasteredTargetFill), targetCount, excluded, contextModel.PreferredDurationSec, s.now())

	gap = targetCount - len(filled)
	if gap <= 0 {
		return filled, nil
	}

	popular, err := s.reader.ListPopularFillCandidates(ctx, contextModel.Request.UserID, videoFillExcludedList(excluded), videoFillQueryLimit(gap))
	if err != nil {
		return nil, err
	}
	filled = appendFillCandidates(filled, popular, string(policy.LanePopularFill), targetCount, excluded, contextModel.PreferredDurationSec, s.now())

	return filled, nil
}

func appendFillCandidates(
	selected []model.VideoCandidate,
	candidates []model.VideoFillCandidate,
	lane string,
	targetCount int,
	excluded map[string]struct{},
	preferredDurationSec [2]int,
	now time.Time,
) []model.VideoCandidate {
	if len(candidates) == 0 || len(selected) >= targetCount {
		return selected
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		left := fillScore(candidates[i], lane, preferredDurationSec, now)
		right := fillScore(candidates[j], lane, preferredDurationSec, now)
		if left != right {
			return left > right
		}
		return candidates[i].VideoID < candidates[j].VideoID
	})

	for _, candidate := range candidates {
		if len(selected) >= targetCount {
			break
		}
		if _, exists := excluded[candidate.VideoID]; exists {
			continue
		}
		selected = append(selected, model.VideoCandidate{
			VideoID:               candidate.VideoID,
			DurationMs:            candidate.DurationMs,
			LaneSources:           []string{lane},
			LearningUnits:         []model.ExpectedLearningUnit{},
			CoverageStrengthScore: round4(fillCoverageScore(candidate)),
			EducationalFitScore:   round4(fillDurationFit(candidate.DurationMs, preferredDurationSec)),
			FreshnessScore:        round4(fillFreshnessScore(candidate, now)),
			RecentServedPenalty:   fillRecentServedPenalty(candidate, now),
			RecentWatchedPenalty:  fillRecentWatchedPenalty(candidate, now),
			BaseScore:             fillScore(candidate, lane, preferredDurationSec, now),
		})
		excluded[candidate.VideoID] = struct{}{}
	}
	return selected
}

func fillScore(candidate model.VideoFillCandidate, lane string, preferredDurationSec [2]int, now time.Time) float64 {
	popularity := fillPopularityScore(candidate)
	freshness := fillFreshnessScore(candidate, now)
	durationFit := fillDurationFit(candidate.DurationMs, preferredDurationSec)
	servedPenalty := fillRecentServedPenalty(candidate, now)
	watchedPenalty := fillRecentWatchedPenalty(candidate, now)

	if lane == string(policy.LaneMasteredTargetFill) {
		return round4(
			0.45*fillTargetRelevanceScore(candidate) +
				0.25*popularity +
				0.20*freshness +
				0.10*durationFit -
				0.08*servedPenalty -
				0.12*watchedPenalty,
		)
	}
	return round4(
		0.60*popularity +
			0.25*freshness +
			0.15*durationFit -
			0.08*servedPenalty -
			0.12*watchedPenalty,
	)
}

func fillTargetRelevanceScore(candidate model.VideoFillCandidate) float64 {
	unitScore := math.Min(float64(candidate.MatchedUnitCount)/4.0, 1.0)
	mentionScore := math.Min(float64(candidate.TotalMentionCount)/12.0, 1.0)
	return clamp01(0.45*unitScore + 0.25*mentionScore + 0.15*candidate.MaxCoverageRatio + 0.15*candidate.MappedSpanRatio)
}

func fillPopularityScore(candidate model.VideoFillCandidate) float64 {
	score := math.Log1p(float64(candidate.ViewCount))*0.45 +
		math.Log1p(float64(candidate.LikeCount))*0.30 +
		math.Log1p(float64(candidate.FavoriteCount))*0.25
	return clamp01(score / 10.0)
}

func fillCoverageScore(candidate model.VideoFillCandidate) float64 {
	return clamp01(0.60*candidate.MappedSpanRatio + 0.40*candidate.MaxCoverageRatio)
}

func fillFreshnessScore(candidate model.VideoFillCandidate, now time.Time) float64 {
	freshness := 1.0
	if penalty := fillRecentServedPenalty(candidate, now); penalty > 0 {
		freshness -= penalty * 0.55
	}
	if penalty := fillRecentWatchedPenalty(candidate, now); penalty > 0 {
		freshness -= penalty * 0.35
	}
	return math.Max(0, freshness)
}

func fillRecentServedPenalty(candidate model.VideoFillCandidate, now time.Time) float64 {
	if candidate.LastServedAt == nil {
		return 0
	}
	recency := fillRecencyPenalty(now.Sub(*candidate.LastServedAt), 72*time.Hour)
	countFactor := math.Min(float64(candidate.ServedCount)/5.0, 1.0)
	return round4(math.Min(1.0, recency*0.70+countFactor*0.30))
}

func fillRecentWatchedPenalty(candidate model.VideoFillCandidate, now time.Time) float64 {
	recency := 0.0
	if candidate.LastWatchedAt != nil {
		recency = fillRecencyPenalty(now.Sub(*candidate.LastWatchedAt), 96*time.Hour)
	}
	countFactor := math.Min(float64(candidate.WatchCount)/5.0, 1.0)
	completionFactor := math.Min(float64(candidate.CompletedCount)/3.0, 1.0)
	return round4(math.Min(1.0, recency*0.45+countFactor*0.25+completionFactor*0.20+fillWatchedRatio(candidate)*0.10))
}

func fillWatchedRatio(candidate model.VideoFillCandidate) float64 {
	if candidate.MaxPositionMs <= 0 || candidate.DurationMs <= 0 {
		return 0
	}
	return math.Min(1.0, math.Max(0, float64(candidate.MaxPositionMs)/float64(candidate.DurationMs)))
}

func fillRecencyPenalty(delta time.Duration, horizon time.Duration) float64 {
	if delta <= 0 {
		return 1.0
	}
	if delta >= horizon {
		return 0
	}
	return 1.0 - float64(delta)/float64(horizon)
}

func fillDurationFit(durationMs int32, preferredDurationSec [2]int) float64 {
	if preferredDurationSec[0] <= 0 || preferredDurationSec[1] <= 0 {
		return 1.0
	}
	minMs := preferredDurationSec[0] * 1000
	maxMs := preferredDurationSec[1] * 1000
	if int(durationMs) >= minMs && int(durationMs) <= maxMs {
		return 1.0
	}
	if int(durationMs) < minMs {
		return math.Max(0.0, 1.0-float64(minMs-int(durationMs))/float64(minMs))
	}
	return math.Max(0.0, 1.0-float64(int(durationMs)-maxMs)/float64(maxMs))
}

func videoFillQueryLimit(gap int) int32 {
	if gap <= 0 {
		return 0
	}
	limit := int32(gap * 3)
	if limit > maxVideoFillQueryLimit {
		return maxVideoFillQueryLimit
	}
	return limit
}

func videoFillExcludedSet(videos []model.VideoCandidate) map[string]struct{} {
	result := make(map[string]struct{}, len(videos))
	for _, video := range videos {
		result[video.VideoID] = struct{}{}
	}
	return result
}

func videoFillExcludedList(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
