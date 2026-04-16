//go:build e2e

package e2e

import (
	"slices"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	recommendationdto "learning-video-recommendation-system/internal/recommendation/application/dto"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func targetSpec(unitID int64, priority float64, ref string) learningdto.TargetUnitSpec {
	return learningdto.TargetUnitSpec{
		CoarseUnitID:      unitID,
		TargetSource:      "e2e",
		TargetSourceRefID: ref,
		TargetPriority:    priority,
	}
}

func singleUnitVideo(videoID string, unitID int64, startMs, endMs int32, sentenceIndex int32, surface string, durationMs int32, mentionCount int32, coverageRatio float64, mappedSpanRatio float64) testutil.CatalogVideoFixture {
	return testutil.CatalogVideoFixture{
		VideoID:          videoID,
		DurationMs:       durationMs,
		Status:           "active",
		VisibilityStatus: "public",
		MappedSpanRatio:  mappedSpanRatio,
		TranscriptEntries: []testutil.TranscriptSentenceFixture{
			{SentenceIndex: sentenceIndex, Text: surface + " core", StartMs: startMs, EndMs: endMs},
			{SentenceIndex: sentenceIndex + 1, Text: surface + " support", StartMs: endMs, EndMs: endMs + 1_000},
		},
		SemanticSpans: []testutil.SemanticSpanFixture{
			{SentenceIndex: sentenceIndex, SpanIndex: 0, CoarseUnitID: &unitID, StartMs: startMs, EndMs: endMs, Text: surface},
		},
		UnitIndexes: []testutil.VideoUnitIndexFixture{
			{
				CoarseUnitID:       unitID,
				MentionCount:       mentionCount,
				SentenceCount:      2,
				FirstStartMs:       startMs,
				LastEndMs:          endMs + 1_000,
				CoverageMs:         endMs + 1_000 - startMs,
				CoverageRatio:      coverageRatio,
				SentenceIndexes:    []int32{sentenceIndex, sentenceIndex + 1},
				EvidenceSpanRefs:   []testutil.EvidenceRefFixture{{SentenceIndex: sentenceIndex, SpanIndex: 0}},
				SampleSurfaceForms: []string{surface},
			},
		},
	}
}

func bundleVideo(videoID string, entries []testutil.VideoUnitIndexFixture, spans []testutil.SemanticSpanFixture, sentences []testutil.TranscriptSentenceFixture, durationMs int32, mappedSpanRatio float64) testutil.CatalogVideoFixture {
	return testutil.CatalogVideoFixture{
		VideoID:           videoID,
		DurationMs:        durationMs,
		Status:            "active",
		VisibilityStatus:  "public",
		MappedSpanRatio:   mappedSpanRatio,
		TranscriptEntries: sentences,
		SemanticSpans:     spans,
		UnitIndexes:       entries,
	}
}

func videoIDs(videos []recommendationdto.RecommendationVideo) []string {
	result := make([]string, 0, len(videos))
	for _, video := range videos {
		result = append(result, video.VideoID)
	}
	return result
}

func videoIndex(videos []recommendationdto.RecommendationVideo, videoID string) int {
	for i, video := range videos {
		if video.VideoID == videoID {
			return i
		}
	}
	return -1
}

func containsUnit(units []int64, unitID int64) bool {
	return slices.Contains(units, unitID)
}

func containsReason(reasons []string, reason string) bool {
	return slices.Contains(reasons, reason)
}

func mustTimeAdd(base time.Time, delta time.Duration) time.Time {
	return base.UTC().Add(delta)
}
