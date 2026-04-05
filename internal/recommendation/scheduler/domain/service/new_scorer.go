package service

import (
	"time"

	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
)

type NewScorer interface {
	Score(candidate appquery.NewCandidate, now time.Time) appquery.ScoredNewCandidate
}

type newScorer struct{}

func NewNewScorer() NewScorer {
	return newScorer{}
}

func (newScorer) Score(candidate appquery.NewCandidate, now time.Time) appquery.ScoredNewCandidate {
	freshnessScore := newFreshnessScore(candidate)
	notRecentlyRecommended := newNotRecentlyRecommended(candidate, now)

	score := 0.75*candidate.State.TargetPriority + 0.15*freshnessScore + 0.10*notRecentlyRecommended

	reasonCodes := []string{"new_candidate"}
	switch freshnessScore {
	case 1:
		reasonCodes = append(reasonCodes, "fresh_candidate")
	case 0.5:
		reasonCodes = append(reasonCodes, "recommended_unconsumed")
	}
	if notRecentlyRecommended == 1 {
		reasonCodes = append(reasonCodes, "not_recently_recommended")
	} else {
		reasonCodes = append(reasonCodes, "recently_recommended")
	}

	return appquery.ScoredNewCandidate{
		Candidate:   candidate,
		Score:       score,
		ReasonCodes: reasonCodes,
	}
}

func newFreshnessScore(candidate appquery.NewCandidate) float64 {
	switch {
	case candidate.State.SeenCount == 0 && candidate.State.StrongEventCount == 0:
		return 1
	case candidate.State.StrongEventCount == 0:
		return 0.5
	default:
		return 0
	}
}

func newNotRecentlyRecommended(candidate appquery.NewCandidate, now time.Time) float64 {
	if candidate.Serving.LastRecommendedAt == nil {
		return 1
	}

	if now.Sub(*candidate.Serving.LastRecommendedAt) >= 24*time.Hour {
		return 1
	}

	return 0
}
