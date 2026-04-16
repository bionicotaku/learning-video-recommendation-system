package service

import (
	"context"
	"encoding/json"
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

func (r *DefaultEvidenceResolver) Resolve(recommendationContext model.RecommendationContext, candidates []model.VideoUnitCandidate, demand model.DemandBundle) ([]model.ResolvedEvidenceWindow, error) {
	_ = recommendationContext
	_ = demand

	resolved := make([]model.ResolvedEvidenceWindow, 0, len(candidates))
	for _, candidate := range candidates {
		spans, err := r.spanReader.ListByVideoAndUnit(context.Background(), candidate.VideoID, candidate.CoarseUnitID)
		if err != nil {
			return nil, err
		}

		refs := parseEvidenceRefs(candidate.EvidenceSpanRefs)
		bestRef, bestSpan := selectBestEvidence(refs, spans)
		if bestSpan == nil && len(spans) > 0 {
			bestSpan = earliestSpan(spans)
			bestRef = &model.EvidenceRef{
				SentenceIndex: bestSpan.SentenceIndex,
				SpanIndex:     bestSpan.SpanIndex,
			}
		}

		windowSentenceIndexes := resolveWindowSentenceIndexes(candidate.SentenceIndexes, refs, bestSpan)
		sentences, err := r.sentenceReader.ListByVideoAndIndexes(context.Background(), candidate.VideoID, windowSentenceIndexes)
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
			ResolvedSpans:         spans,
			ResolvedSentences:     sentences,
		})
	}

	return resolved, nil
}

func parseEvidenceRefs(raw []byte) []model.EvidenceRef {
	if len(raw) == 0 {
		return nil
	}

	var refs []model.EvidenceRef
	if err := json.Unmarshal(raw, &refs); err != nil {
		return nil
	}
	return refs
}

func selectBestEvidence(refs []model.EvidenceRef, spans []model.SemanticSpan) (*model.EvidenceRef, *model.SemanticSpan) {
	if len(spans) == 0 {
		return nil, nil
	}

	type matchedRef struct {
		refIndex int
		ref      model.EvidenceRef
		span     model.SemanticSpan
	}

	spanByKey := make(map[[2]int32]model.SemanticSpan, len(spans))
	for _, span := range spans {
		spanByKey[[2]int32{span.SentenceIndex, span.SpanIndex}] = span
	}

	matches := make([]matchedRef, 0, len(refs))
	for index, ref := range refs {
		span, ok := spanByKey[[2]int32{ref.SentenceIndex, ref.SpanIndex}]
		if !ok {
			continue
		}
		matches = append(matches, matchedRef{refIndex: index, ref: ref, span: span})
	}
	if len(matches) == 0 {
		return nil, nil
	}

	best := matches[0]
	for _, candidate := range matches[1:] {
		if candidate.ref.SentenceIndex == best.ref.SentenceIndex {
			if candidate.span.StartMs < best.span.StartMs {
				best = candidate
				continue
			}
			if candidate.span.StartMs == best.span.StartMs && candidate.refIndex < best.refIndex {
				best = candidate
			}
			continue
		}
		if candidate.refIndex < best.refIndex {
			best = candidate
		}
	}

	ref := best.ref
	span := best.span
	return &ref, &span
}

func earliestSpan(spans []model.SemanticSpan) *model.SemanticSpan {
	if len(spans) == 0 {
		return nil
	}

	best := spans[0]
	for _, candidate := range spans[1:] {
		if candidate.StartMs < best.StartMs {
			best = candidate
			continue
		}
		if candidate.StartMs == best.StartMs && candidate.SentenceIndex < best.SentenceIndex {
			best = candidate
		}
	}
	return &best
}

func resolveWindowSentenceIndexes(candidateSentenceIndexes []int32, refs []model.EvidenceRef, bestSpan *model.SemanticSpan) []int32 {
	indexes := appendUniqueInt32(nil, candidateSentenceIndexes...)
	if len(indexes) == 0 {
		for _, ref := range refs {
			indexes = appendUniqueInt32(indexes, ref.SentenceIndex)
		}
	}
	if len(indexes) == 0 && bestSpan != nil {
		indexes = append(indexes, bestSpan.SentenceIndex)
	}

	sort.SliceStable(indexes, func(i, j int) bool {
		return indexes[i] < indexes[j]
	})
	return indexes
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

func appendUniqueInt32(values []int32, additions ...int32) []int32 {
	seen := make(map[int32]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}
