package service

import (
	"context"
	"sort"
	"strconv"

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

	bestSpans, err := r.loadBestSpans(ctx, candidates)
	if err != nil {
		return nil, err
	}
	sentencesByKey, err := r.loadWindowSentences(ctx, bestSpans)
	if err != nil {
		return nil, err
	}

	resolved := make([]model.ResolvedEvidenceWindow, 0, len(candidates))
	for _, candidate := range candidates {
		bestRef := candidate.BestEvidenceRef
		bestSpan := bestSpans[bestSpanKey(candidate.VideoID, candidate.CoarseUnitID, candidate.BestEvidenceRef)]
		if bestSpan == nil {
			bestRef = nil
		}

		windowSentenceIndexes := resolveWindowSentenceIndexes(bestSpan)
		sentences := sentencesForWindow(candidate.VideoID, windowSentenceIndexes, sentencesByKey)
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

func (r *DefaultEvidenceResolver) loadBestSpans(ctx context.Context, candidates []model.VideoUnitCandidate) (map[string]*model.SemanticSpan, error) {
	refs := make([]apprepo.SemanticSpanRef, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate.BestEvidenceRef == nil {
			continue
		}
		key := bestSpanKey(candidate.VideoID, candidate.CoarseUnitID, candidate.BestEvidenceRef)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, apprepo.SemanticSpanRef{
			VideoID:      candidate.VideoID,
			CoarseUnitID: candidate.CoarseUnitID,
			Ref:          *candidate.BestEvidenceRef,
		})
	}
	if len(refs) == 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return map[string]*model.SemanticSpan{}, nil
	}

	rows, err := r.spanReader.ListByVideoUnitRefs(ctx, refs)
	if err != nil {
		return nil, err
	}

	spans := make(map[string]*model.SemanticSpan, len(rows))
	for _, row := range rows {
		if row.CoarseUnitID == nil {
			continue
		}
		rowCopy := row
		ref := &model.EvidenceRef{SentenceIndex: row.SentenceIndex, SpanIndex: row.SpanIndex}
		spans[bestSpanKey(row.VideoID, *row.CoarseUnitID, ref)] = &rowCopy
	}
	return spans, nil
}

func (r *DefaultEvidenceResolver) loadWindowSentences(ctx context.Context, bestSpans map[string]*model.SemanticSpan) (map[string]model.TranscriptSentence, error) {
	refs := make([]apprepo.TranscriptSentenceRef, 0, len(bestSpans))
	seen := make(map[string]struct{}, len(bestSpans))
	for _, span := range bestSpans {
		if span == nil {
			continue
		}
		for _, sentenceIndex := range resolveWindowSentenceIndexes(span) {
			key := sentenceKey(span.VideoID, sentenceIndex)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			refs = append(refs, apprepo.TranscriptSentenceRef{
				VideoID:       span.VideoID,
				SentenceIndex: sentenceIndex,
			})
		}
	}
	if len(refs) == 0 {
		return map[string]model.TranscriptSentence{}, nil
	}

	rows, err := r.sentenceReader.ListByVideoAndIndexesBatch(ctx, refs)
	if err != nil {
		return nil, err
	}

	sentences := make(map[string]model.TranscriptSentence, len(rows))
	for _, row := range rows {
		sentences[sentenceKey(row.VideoID, row.SentenceIndex)] = row
	}
	return sentences, nil
}

func bestSpanKey(videoID string, coarseUnitID int64, ref *model.EvidenceRef) string {
	if ref == nil {
		return ""
	}
	return videoID + "#" + int64Key(coarseUnitID) + "#" + int32Key(ref.SentenceIndex) + "#" + int32Key(ref.SpanIndex)
}

func sentenceKey(videoID string, sentenceIndex int32) string {
	return videoID + "#" + int32Key(sentenceIndex)
}

func sentencesForWindow(videoID string, sentenceIndexes []int32, sentencesByKey map[string]model.TranscriptSentence) []model.TranscriptSentence {
	result := make([]model.TranscriptSentence, 0, len(sentenceIndexes))
	for _, sentenceIndex := range sentenceIndexes {
		if sentence, ok := sentencesByKey[sentenceKey(videoID, sentenceIndex)]; ok {
			result = append(result, sentence)
		}
	}
	return result
}

func int64Key(value int64) string {
	return strconv.FormatInt(value, 10)
}

func int32Key(value int32) string {
	return strconv.FormatInt(int64(value), 10)
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
