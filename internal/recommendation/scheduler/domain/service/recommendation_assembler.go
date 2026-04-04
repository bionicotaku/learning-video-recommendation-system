package service

import (
	"sort"
	"time"

	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

	"github.com/google/uuid"
)

type RecommendationAssembler interface {
	Assemble(userID uuid.UUID, generatedAt time.Time, priorityZero []appquery.ScoredReviewCandidate, scoredReviews []appquery.ScoredReviewCandidate, scoredNews []appquery.ScoredNewCandidate, quotas QuotaAllocation) model.RecommendationBatch
}

type recommendationAssembler struct{}

func NewRecommendationAssembler() RecommendationAssembler {
	return recommendationAssembler{}
}

func (recommendationAssembler) Assemble(userID uuid.UUID, generatedAt time.Time, priorityZero []appquery.ScoredReviewCandidate, scoredReviews []appquery.ScoredReviewCandidate, scoredNews []appquery.ScoredNewCandidate, quotas QuotaAllocation) model.RecommendationBatch {
	sessionLimit := quotas.ReviewQuota + quotas.NewQuota
	items := make([]model.RecommendationItem, 0, sessionLimit)
	seen := make(map[int64]struct{})

	priorityZero = append([]appquery.ScoredReviewCandidate(nil), priorityZero...)
	scoredReviews = append([]appquery.ScoredReviewCandidate(nil), scoredReviews...)
	scoredNews = append([]appquery.ScoredNewCandidate(nil), scoredNews...)

	sortScoredReviewCandidates(scoredReviews)
	sortScoredNewCandidates(scoredNews)

	reviewBudget := quotas.ReviewQuota
	for _, item := range priorityZero {
		if reviewBudget == 0 {
			break
		}
		if _, ok := seen[item.Candidate.State.CoarseUnitID]; ok {
			continue
		}
		items = append(items, reviewRecommendationItem(item))
		seen[item.Candidate.State.CoarseUnitID] = struct{}{}
		reviewBudget--
	}

	for _, item := range scoredReviews {
		if reviewBudget == 0 {
			break
		}
		if _, ok := seen[item.Candidate.State.CoarseUnitID]; ok {
			continue
		}
		items = append(items, reviewRecommendationItem(item))
		seen[item.Candidate.State.CoarseUnitID] = struct{}{}
		reviewBudget--
	}

	remainingSlots := sessionLimit - len(items)
	for _, item := range scoredNews {
		if remainingSlots == 0 {
			break
		}
		if _, ok := seen[item.Candidate.State.CoarseUnitID]; ok {
			continue
		}
		items = append(items, newRecommendationItem(item))
		seen[item.Candidate.State.CoarseUnitID] = struct{}{}
		remainingSlots--
	}

	for index := range items {
		items[index].Rank = index + 1
	}

	return model.RecommendationBatch{
		RunID:             uuid.New(),
		UserID:            userID,
		GeneratedAt:       generatedAt,
		SessionLimit:      sessionLimit,
		ReviewQuota:       quotas.ReviewQuota,
		NewQuota:          quotas.NewQuota,
		BacklogProtection: quotas.BacklogProtection,
		Items:             items,
	}
}

func sortScoredReviewCandidates(items []appquery.ScoredReviewCandidate) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			return items[i].Candidate.State.CoarseUnitID < items[j].Candidate.State.CoarseUnitID
		}

		return items[i].Score > items[j].Score
	})
}

func sortScoredNewCandidates(items []appquery.ScoredNewCandidate) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			return items[i].Candidate.State.CoarseUnitID < items[j].Candidate.State.CoarseUnitID
		}

		return items[i].Score > items[j].Score
	})
}

func reviewRecommendationItem(item appquery.ScoredReviewCandidate) model.RecommendationItem {
	return model.RecommendationItem{
		CoarseUnitID:    item.Candidate.State.CoarseUnitID,
		Kind:            item.Candidate.Unit.Kind,
		Label:           item.Candidate.Unit.Label,
		RecommendType:   enum.RecommendTypeReview,
		Status:          item.Candidate.State.Status,
		Score:           item.Score,
		ReasonCodes:     uniqueStrings(item.ReasonCodes),
		TargetPriority:  item.Candidate.State.TargetPriority,
		ProgressPercent: item.Candidate.State.ProgressPercent,
		MasteryScore:    item.Candidate.State.MasteryScore,
		NextReviewAt:    item.Candidate.State.NextReviewAt,
	}
}

func newRecommendationItem(item appquery.ScoredNewCandidate) model.RecommendationItem {
	return model.RecommendationItem{
		CoarseUnitID:    item.Candidate.State.CoarseUnitID,
		Kind:            item.Candidate.Unit.Kind,
		Label:           item.Candidate.Unit.Label,
		RecommendType:   enum.RecommendTypeNew,
		Status:          item.Candidate.State.Status,
		Score:           item.Score,
		ReasonCodes:     uniqueStrings(item.ReasonCodes),
		TargetPriority:  item.Candidate.State.TargetPriority,
		ProgressPercent: item.Candidate.State.ProgressPercent,
		MasteryScore:    item.Candidate.State.MasteryScore,
		NextReviewAt:    item.Candidate.State.NextReviewAt,
	}
}
