package service

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	domainresolver "learning-video-recommendation-system/internal/recommendation/domain/resolver"
)

type DefaultEvidenceResolver struct {
}

var _ domainresolver.EvidenceResolver = (*DefaultEvidenceResolver)(nil)

func NewDefaultEvidenceResolver() *DefaultEvidenceResolver {
	return &DefaultEvidenceResolver{}
}

func (r *DefaultEvidenceResolver) Resolve(ctx context.Context, recommendationContext model.RecommendationContext, candidates []model.VideoUnitCandidate, demand model.DemandBundle) ([]model.ResolvedEvidenceWindow, error) {
	_ = recommendationContext
	_ = demand
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	resolved := make([]model.ResolvedEvidenceWindow, 0, len(candidates))
	for _, candidate := range candidates {
		bestRef := candidate.BestEvidenceRef
		bestStartMs, bestEndMs := resolveCandidateBounds(candidate)
		windowSentenceIndexes := resolveCandidateWindowSentenceIndexes(bestRef)

		resolved = append(resolved, model.ResolvedEvidenceWindow{
			Candidate:             candidate,
			BestEvidenceRef:       bestRef,
			BestEvidenceStartMs:   bestStartMs,
			BestEvidenceEndMs:     bestEndMs,
			WindowSentenceIndexes: windowSentenceIndexes,
			WindowStartMs:         bestStartMs,
			WindowEndMs:           bestEndMs,
		})
	}

	return resolved, nil
}

func resolveCandidateBounds(candidate model.VideoUnitCandidate) (*int32, *int32) {
	if candidate.BestEvidenceStartMs <= 0 && candidate.BestEvidenceEndMs <= 0 {
		return nil, nil
	}
	startMs := candidate.BestEvidenceStartMs
	endMs := candidate.BestEvidenceEndMs
	return &startMs, &endMs
}

func resolveCandidateWindowSentenceIndexes(ref *model.EvidenceRef) []int32 {
	if ref == nil {
		return nil
	}
	return []int32{ref.SentenceIndex}
}
