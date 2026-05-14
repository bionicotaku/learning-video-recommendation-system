package service

import (
	"context"
	"sort"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	domainresolver "learning-video-recommendation-system/internal/recommendation/domain/resolver"
)

type DefaultEvidenceResolver struct {
	spanReader     apprepo.SemanticSpanReader
	sentenceReader apprepo.TranscriptSentenceReader
}

var _ domainresolver.EvidenceResolver = (*DefaultEvidenceResolver)(nil)

func NewDefaultEvidenceResolver(
	spanReader apprepo.SemanticSpanReader,
	sentenceReader apprepo.TranscriptSentenceReader,
) *DefaultEvidenceResolver {
	return &DefaultEvidenceResolver{
		spanReader:     spanReader,
		sentenceReader: sentenceReader,
	}
}

func (r *DefaultEvidenceResolver) Resolve(ctx context.Context, recommendationContext model.RecommendationContext, candidates []model.VideoUnitCandidate, demand model.DemandBundle) ([]model.ResolvedEvidenceWindow, error) {
	_ = recommendationContext
	_ = demand

	resolved := make([]model.ResolvedEvidenceWindow, 0, len(candidates))
	for _, candidate := range candidates {
		bestRef := candidate.BestEvidenceRef
		bestSpan, err := r.resolveBestSpan(ctx, candidate)
		if err != nil {
			return nil, err
		}
		if bestSpan == nil {
			bestRef = nil
		}

		windowSentenceIndexes := resolveWindowSentenceIndexes(bestSpan)
		sentences, err := r.sentenceReader.ListByVideoAndIndexes(ctx, candidate.VideoID, windowSentenceIndexes)
		if err != nil {
			return nil, err
		}
		sort.SliceStable(sentences, func(i, j int) bool {
			return sentences[i].SentenceIndex < sentences[j].SentenceIndex
		})

		windowStartMs, windowEndMs := resolveWindowBounds(sentences, bestSpan)
		bestStartMs, bestEndMs := resolveBestBounds(bestSpan)

		resolved = append(resolved, model.ResolvedEvidenceWindow{
			Candidate:             candidate,
			BestEvidenceRef:       bestRef,
			BestEvidenceStartMs:   bestStartMs,
			BestEvidenceEndMs:     bestEndMs,
			WindowSentenceIndexes: windowSentenceIndexes,
			WindowStartMs:         windowStartMs,
			WindowEndMs:           windowEndMs,
			ResolvedSpans:         resolvedSpans(bestSpan),
			ResolvedSentences:     sentences,
		})
	}

	return resolved, nil
}

func (r *DefaultEvidenceResolver) resolveBestSpan(ctx context.Context, candidate model.VideoUnitCandidate) (*model.SemanticSpan, error) {
	if candidate.BestEvidenceRef == nil {
		return nil, nil
	}
	return r.spanReader.GetByVideoUnitAndRef(ctx, candidate.VideoID, candidate.CoarseUnitID, *candidate.BestEvidenceRef)
}

func resolveWindowSentenceIndexes(bestSpan *model.SemanticSpan) []int32 {
	if bestSpan == nil {
		return nil
	}
	return []int32{bestSpan.SentenceIndex}
}

func resolveWindowBounds(sentences []model.TranscriptSentence, bestSpan *model.SemanticSpan) (*int32, *int32) {
	if len(sentences) > 0 {
		startMs := sentences[0].StartMs
		endMs := sentences[len(sentences)-1].EndMs
		return &startMs, &endMs
	}
	return resolveBestBounds(bestSpan)
}

func resolveBestBounds(bestSpan *model.SemanticSpan) (*int32, *int32) {
	if bestSpan == nil {
		return nil, nil
	}

	startMs := bestSpan.StartMs
	endMs := bestSpan.EndMs
	return &startMs, &endMs
}

func resolvedSpans(bestSpan *model.SemanticSpan) []model.SemanticSpan {
	if bestSpan == nil {
		return nil
	}
	return []model.SemanticSpan{*bestSpan}
}
