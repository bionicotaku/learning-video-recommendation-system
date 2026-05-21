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

const (
	minRecallScopeUnits = 64
	maxRecallScopeUnits = 200
	maxNoSupplyScopeCap = 8
)

type RecallQueueService struct {
	repository apprepo.RecallQueueRepository
}

func NewRecallQueueService(repository apprepo.RecallQueueRepository) *RecallQueueService {
	return &RecallQueueService{repository: repository}
}

func (s *RecallQueueService) SelectScope(ctx context.Context, userID string, targetVideoCount int, now time.Time) (model.RecallScopeSelection, error) {
	scopeLimit := recallScopeLimit(targetVideoCount)
	noSupplyLimit := maxNoSupplyScopeUnits(scopeLimit)
	summary := model.RecallScopeSummary{
		PlannerScopeUnitCountByBucket: map[string]int{},
		PerUnitRecallLimit:            recallRowsPerUnitLimitForTarget(targetVideoCount),
	}
	if s == nil || s.repository == nil {
		return model.RecallScopeSelection{Summary: summary}, nil
	}

	projectionUpdatedAt, err := s.repository.GetProjectionUpdatedAt(ctx)
	if err != nil {
		return model.RecallScopeSelection{}, err
	}
	learningVersion, err := s.repository.GetLearningStateVersion(ctx, userID)
	if err != nil {
		return model.RecallScopeSelection{}, err
	}
	summary.ActiveTargetUnitCount = int(learningVersion.ActiveTargetUnitCount)

	state, exists, err := s.repository.GetQueueState(ctx, userID)
	if err != nil {
		return model.RecallScopeSelection{}, err
	}
	if !exists || queueStale(state, learningVersion, projectionUpdatedAt) {
		state, err = s.repository.RebuildUserQueue(ctx, userID, projectionUpdatedAt)
		if err != nil {
			return model.RecallScopeSelection{}, err
		}
		summary.QueueRebuilt = true
		summary.ActiveTargetUnitCount = int(state.ActiveTargetUnitCount)
	}

	candidates, err := s.repository.ListCandidates(ctx, userID, now, int32(scopeLimit), int32(noSupplyLimit))
	if err != nil {
		return model.RecallScopeSelection{}, err
	}
	summary.QueueCandidateCount = len(candidates)

	selected := selectRecallScope(candidates, scopeLimit)
	recallFetchScope := recallFetchScope(selected)
	summary.PlannerScopeUnitCount = len(selected)
	for _, candidate := range selected {
		summary.PlannerScopeUnitCountByBucket[candidate.Bucket]++
		if candidate.SupplyGrade == "none" {
			summary.NoSupplyScopeUnitCount++
		}
	}
	summary.RecallFetchUnitCount = len(recallFetchScope)
	summary.MaxPossibleRecallRows = summary.RecallFetchUnitCount * int(summary.PerUnitRecallLimit)
	return model.RecallScopeSelection{
		PlannerScope:     selected,
		RecallFetchScope: recallFetchScope,
		Summary:          summary,
	}, nil
}

func queueStale(state model.RecallQueueState, learningVersion apprepo.LearningStateVersion, projectionUpdatedAt time.Time) bool {
	if state.ActiveTargetUnitCount != learningVersion.ActiveTargetUnitCount {
		return true
	}
	if !sameOptionalTime(state.SourceLearningMaxUpdatedAt, learningVersion.SourceLearningMaxUpdatedAt) {
		return true
	}
	return !state.SourceProjectionUpdatedAt.Equal(projectionUpdatedAt)
}

func sameOptionalTime(left *time.Time, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Equal(*right)
	}
}

func recallScopeLimit(targetVideoCount int) int {
	if targetVideoCount <= 0 {
		targetVideoCount = defaultTargetVideoCount
	}
	return minInt(maxInt(targetVideoCount*12, minRecallScopeUnits), maxRecallScopeUnits)
}

func selectRecallScope(candidates []model.RecallQueueCandidate, scopeLimit int) []model.RecallQueueCandidate {
	if scopeLimit <= 0 || len(candidates) == 0 {
		return nil
	}
	byBucket := map[string][]model.RecallQueueCandidate{}
	for _, candidate := range candidates {
		byBucket[candidate.Bucket] = append(byBucket[candidate.Bucket], candidate)
	}
	for bucket := range byBucket {
		sortRecallCandidates(byBucket[bucket])
	}

	quotas := recallScopeQuotas(scopeLimit, len(byBucket[string(policy.BucketHardReview)]))
	selected := make([]model.RecallQueueCandidate, 0, scopeLimit)
	selectedKeys := make(map[int64]struct{}, scopeLimit)
	noSupplySelected := 0
	maxNoSupply := maxNoSupplyScopeUnits(scopeLimit)
	appendSelected := func(candidate model.RecallQueueCandidate) bool {
		if _, ok := selectedKeys[candidate.CoarseUnitID]; ok {
			return false
		}
		if candidate.SupplyGrade == "none" {
			if noSupplySelected >= maxNoSupply {
				return false
			}
			noSupplySelected++
		}
		selected = append(selected, candidate)
		selectedKeys[candidate.CoarseUnitID] = struct{}{}
		return true
	}
	for _, bucket := range []string{
		string(policy.BucketHardReview),
		string(policy.BucketNewNow),
		string(policy.BucketSoftReview),
		string(policy.BucketNearFuture),
	} {
		taken := 0
		for _, candidate := range byBucket[bucket] {
			if taken >= quotas[bucket] {
				break
			}
			if appendSelected(candidate) {
				taken++
			}
		}
	}

	if len(selected) < scopeLimit {
		remaining := make([]model.RecallQueueCandidate, 0, len(candidates)-len(selected))
		for _, candidate := range candidates {
			if _, ok := selectedKeys[candidate.CoarseUnitID]; ok {
				continue
			}
			remaining = append(remaining, candidate)
		}
		sort.SliceStable(remaining, func(i, j int) bool {
			if remaining[i].DynamicPriority != remaining[j].DynamicPriority {
				return remaining[i].DynamicPriority > remaining[j].DynamicPriority
			}
			if bucketPriority(remaining[i].Bucket) != bucketPriority(remaining[j].Bucket) {
				return bucketPriority(remaining[i].Bucket) < bucketPriority(remaining[j].Bucket)
			}
			return remaining[i].CoarseUnitID < remaining[j].CoarseUnitID
		})
		for _, candidate := range remaining {
			if len(selected) >= scopeLimit {
				break
			}
			appendSelected(candidate)
		}
	}

	sort.SliceStable(selected, func(i, j int) bool {
		if bucketPriority(selected[i].Bucket) != bucketPriority(selected[j].Bucket) {
			return bucketPriority(selected[i].Bucket) < bucketPriority(selected[j].Bucket)
		}
		if selected[i].DynamicPriority != selected[j].DynamicPriority {
			return selected[i].DynamicPriority > selected[j].DynamicPriority
		}
		return selected[i].CoarseUnitID < selected[j].CoarseUnitID
	})
	return selected
}

func maxNoSupplyScopeUnits(scopeLimit int) int {
	if scopeLimit <= 0 {
		return 0
	}
	return minInt(maxNoSupplyScopeCap, maxInt(1, scopeLimit/10))
}

func recallFetchScope(scope []model.RecallQueueCandidate) []model.RecallQueueCandidate {
	result := make([]model.RecallQueueCandidate, 0, len(scope))
	for _, candidate := range scope {
		if candidate.SupplyGrade != "none" {
			result = append(result, candidate)
		}
	}
	return result
}

func recallScopeQuotas(scopeLimit int, hardCount int) map[string]int {
	ratios := map[string]float64{
		string(policy.BucketHardReview): 0.40,
		string(policy.BucketNewNow):     0.30,
		string(policy.BucketSoftReview): 0.20,
		string(policy.BucketNearFuture): 0.10,
	}
	defaultHardQuota := int(math.Floor(float64(scopeLimit) * ratios[string(policy.BucketHardReview)]))
	if hardCount > defaultHardQuota {
		ratios[string(policy.BucketHardReview)] = 0.50
		ratios[string(policy.BucketNewNow)] = 0.25
		ratios[string(policy.BucketSoftReview)] = 0.15
		ratios[string(policy.BucketNearFuture)] = 0.10
	}

	quotas := make(map[string]int, len(ratios))
	assigned := 0
	for _, bucket := range []string{
		string(policy.BucketHardReview),
		string(policy.BucketNewNow),
		string(policy.BucketSoftReview),
		string(policy.BucketNearFuture),
	} {
		quota := int(math.Floor(float64(scopeLimit) * ratios[bucket]))
		quotas[bucket] = quota
		assigned += quota
	}
	for _, bucket := range []string{
		string(policy.BucketHardReview),
		string(policy.BucketNewNow),
		string(policy.BucketSoftReview),
		string(policy.BucketNearFuture),
	} {
		if assigned >= scopeLimit {
			break
		}
		quotas[bucket]++
		assigned++
	}
	return quotas
}

func sortRecallCandidates(candidates []model.RecallQueueCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].DynamicPriority != candidates[j].DynamicPriority {
			return candidates[i].DynamicPriority > candidates[j].DynamicPriority
		}
		return candidates[i].CoarseUnitID < candidates[j].CoarseUnitID
	})
}
