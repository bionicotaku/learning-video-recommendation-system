package service

import (
	"time"

	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
)

type ReviewScorer interface {
	Score(candidate appquery.ReviewCandidate, now time.Time) appquery.ScoredReviewCandidate
}

type reviewScorer struct{}

func NewReviewScorer() ReviewScorer {
	return reviewScorer{}
}

func (reviewScorer) Score(candidate appquery.ReviewCandidate, now time.Time) appquery.ScoredReviewCandidate {
	overdueScore := reviewOverdueScore(candidate, now)
	weakMemoryScore := reviewWeakMemoryScore(candidate)
	recencyAdjustment := reviewRecencyAdjustment(candidate, now)

	score := 0.45*overdueScore + 0.25*candidate.State.TargetPriority + 0.20*weakMemoryScore + 0.10*recencyAdjustment

	reasonCodes := []string{"review_due"}
	if overdueScore > 0 {
		reasonCodes = append(reasonCodes, "overdue")
	}
	if weakMemoryScore > 0 {
		reasonCodes = append(reasonCodes, "weak_memory")
	}
	if candidate.State.LastQuality != nil && *candidate.State.LastQuality <= 2 {
		reasonCodes = append(reasonCodes, "recent_failure")
	}
	if recencyAdjustment == 1 {
		reasonCodes = append(reasonCodes, "not_recently_recommended")
	} else {
		reasonCodes = append(reasonCodes, "recently_recommended")
	}

	return appquery.ScoredReviewCandidate{
		Candidate:   candidate,
		Score:       score,
		ReasonCodes: reasonCodes,
	}
}

func reviewOverdueScore(candidate appquery.ReviewCandidate, now time.Time) float64 {
	if candidate.State.NextReviewAt == nil || !now.After(*candidate.State.NextReviewAt) {
		return 0
	}

	overdue := now.Sub(*candidate.State.NextReviewAt)
	if overdue >= 72*time.Hour {
		return 1
	}

	return overdue.Hours() / 72.0
}

func reviewWeakMemoryScore(candidate appquery.ReviewCandidate) float64 {
	qualityPenalty := 0.0
	if candidate.State.LastQuality != nil && *candidate.State.LastQuality <= 2 {
		qualityPenalty = 1
	}

	score := 0.5*(1-candidate.State.MasteryScore) + 0.3*(float64(candidate.State.ConsecutiveWrong)/3.0) + 0.2*qualityPenalty
	if score > 1 {
		return 1
	}
	if score < 0 {
		return 0
	}

	return score
}

func reviewRecencyAdjustment(candidate appquery.ReviewCandidate, now time.Time) float64 {
	if candidate.Serving.LastRecommendedAt == nil {
		return 1
	}

	if now.Sub(*candidate.Serving.LastRecommendedAt) >= 6*time.Hour {
		return 1
	}

	return 0
}
