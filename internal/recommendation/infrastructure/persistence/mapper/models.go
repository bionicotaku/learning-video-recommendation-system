package mapper

import (
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

func ToLearningStateSnapshot(row recommendationsqlc.LearningUserUnitState) (model.LearningStateSnapshot, error) {
	targetPriority, err := NumericToFloat64(row.TargetPriority)
	if err != nil {
		return model.LearningStateSnapshot{}, err
	}
	progressPercent, err := NumericToFloat64(row.ProgressPercent)
	if err != nil {
		return model.LearningStateSnapshot{}, err
	}
	masteryScore, err := NumericToFloat64(row.MasteryScore)
	if err != nil {
		return model.LearningStateSnapshot{}, err
	}

	return model.LearningStateSnapshot{
		UserID:                  UUIDToString(row.UserID),
		CoarseUnitID:            row.CoarseUnitID,
		IsTarget:                row.IsTarget,
		TargetPriority:          targetPriority,
		Status:                  row.Status,
		ProgressPercent:         progressPercent,
		MasteryScore:            masteryScore,
		LastQuality:             Int16PointerFromPG(row.LastQuality),
		NextReviewAt:            TimePointerFromPG(row.NextReviewAt),
		RecentQualityWindow:     row.RecentQualityWindow,
		RecentCorrectnessWindow: row.RecentCorrectnessWindow,
		StrongEventCount:        row.StrongEventCount,
		ReviewCount:             row.ReviewCount,
		UpdatedAt:               row.UpdatedAt.Time,
	}, nil
}

func ToRecommendableVideoUnit(row recommendationsqlc.RecommendationVRecommendableVideoUnit) (model.RecommendableVideoUnit, error) {
	coarseUnitID := int64(0)
	if row.CoarseUnitID.Valid {
		coarseUnitID = row.CoarseUnitID.Int64
	}
	mentionCount := int32(0)
	if row.MentionCount.Valid {
		mentionCount = row.MentionCount.Int32
	}
	sentenceCount := int32(0)
	if row.SentenceCount.Valid {
		sentenceCount = row.SentenceCount.Int32
	}
	firstStartMs := int32(0)
	if row.FirstStartMs.Valid {
		firstStartMs = row.FirstStartMs.Int32
	}
	lastEndMs := int32(0)
	if row.LastEndMs.Valid {
		lastEndMs = row.LastEndMs.Int32
	}
	coverageMs := int32(0)
	if row.CoverageMs.Valid {
		coverageMs = row.CoverageMs.Int32
	}
	coverageRatio, err := NumericToFloat64(row.CoverageRatio)
	if err != nil {
		return model.RecommendableVideoUnit{}, err
	}
	durationMs := int32(0)
	if row.DurationMs.Valid {
		durationMs = row.DurationMs.Int32
	}
	mappedSpanRatio, err := NumericToFloat64(row.MappedSpanRatio)
	if err != nil {
		return model.RecommendableVideoUnit{}, err
	}

	return model.RecommendableVideoUnit{
		VideoID:            UUIDToString(row.VideoID),
		CoarseUnitID:       coarseUnitID,
		MentionCount:       mentionCount,
		SentenceCount:      sentenceCount,
		FirstStartMs:       firstStartMs,
		LastEndMs:          lastEndMs,
		CoverageMs:         coverageMs,
		CoverageRatio:      coverageRatio,
		SentenceIndexes:    row.SentenceIndexes,
		EvidenceSpanRefs:   row.EvidenceSpanRefs,
		SampleSurfaceForms: row.SampleSurfaceForms,
		DurationMs:         durationMs,
		MappedSpanRatio:    mappedSpanRatio,
		Status:             TextToString(row.Status),
		VisibilityStatus:   TextToString(row.VisibilityStatus),
		PublishAt:          TimePointerFromPG(row.PublishAt),
	}, nil
}

func ToUnitVideoInventory(row recommendationsqlc.RecommendationVUnitVideoInventory) (model.UnitVideoInventory, error) {
	coarseUnitID := int64(0)
	if row.CoarseUnitID.Valid {
		coarseUnitID = row.CoarseUnitID.Int64
	}
	distinctVideoCount := int32(0)
	if row.DistinctVideoCount.Valid {
		distinctVideoCount = row.DistinctVideoCount.Int32
	}
	avgMentionCount, err := NumericToFloat64(row.AvgMentionCount)
	if err != nil {
		return model.UnitVideoInventory{}, err
	}
	avgSentenceCount, err := NumericToFloat64(row.AvgSentenceCount)
	if err != nil {
		return model.UnitVideoInventory{}, err
	}
	avgCoverageMs, err := NumericToFloat64(row.AvgCoverageMs)
	if err != nil {
		return model.UnitVideoInventory{}, err
	}
	avgCoverageRatio, err := NumericToFloat64(row.AvgCoverageRatio)
	if err != nil {
		return model.UnitVideoInventory{}, err
	}
	strongVideoCount := int32(0)
	if row.StrongVideoCount.Valid {
		strongVideoCount = row.StrongVideoCount.Int32
	}

	return model.UnitVideoInventory{
		CoarseUnitID:       coarseUnitID,
		DistinctVideoCount: distinctVideoCount,
		AvgMentionCount:    avgMentionCount,
		AvgSentenceCount:   avgSentenceCount,
		AvgCoverageMs:      avgCoverageMs,
		AvgCoverageRatio:   avgCoverageRatio,
		StrongVideoCount:   strongVideoCount,
		SupplyGrade:        TextToString(row.SupplyGrade),
		UpdatedAt:          row.UpdatedAt.Time,
	}, nil
}

func ToSemanticSpan(row recommendationsqlc.CatalogVideoSemanticSpan) model.SemanticSpan {
	return model.SemanticSpan{
		VideoID:       UUIDToString(row.VideoID),
		SentenceIndex: row.SentenceIndex,
		SpanIndex:     row.SpanIndex,
		CoarseUnitID:  Int64PointerFromPG(row.CoarseUnitID),
		StartMs:       row.StartMs,
		EndMs:         row.EndMs,
		Text:          row.Text,
		Explanation:   TextToString(row.Explanation),
	}
}

func ToTranscriptSentence(row recommendationsqlc.CatalogVideoTranscriptSentence) model.TranscriptSentence {
	return model.TranscriptSentence{
		VideoID:       UUIDToString(row.VideoID),
		SentenceIndex: row.SentenceIndex,
		Text:          row.Text,
		StartMs:       row.StartMs,
		EndMs:         row.EndMs,
		Explanation:   TextToString(row.Explanation),
	}
}

func ToVideoUserState(row recommendationsqlc.CatalogVideoUserState) (model.VideoUserState, error) {
	lastWatchRatio, err := NumericToFloat64(row.LastWatchRatio)
	if err != nil {
		return model.VideoUserState{}, err
	}
	maxWatchRatio, err := NumericToFloat64(row.MaxWatchRatio)
	if err != nil {
		return model.VideoUserState{}, err
	}

	return model.VideoUserState{
		UserID:         UUIDToString(row.UserID),
		VideoID:        UUIDToString(row.VideoID),
		LastWatchedAt:  TimePointerFromPG(row.LastWatchedAt),
		WatchCount:     row.WatchCount,
		CompletedCount: row.CompletedCount,
		LastWatchRatio: lastWatchRatio,
		MaxWatchRatio:  maxWatchRatio,
	}, nil
}

func ToUserUnitServingState(row recommendationsqlc.RecommendationUserUnitServingState) model.UserUnitServingState {
	return model.UserUnitServingState{
		UserID:       UUIDToString(row.UserID),
		CoarseUnitID: row.CoarseUnitID,
		LastServedAt: TimePointerFromPG(row.LastServedAt),
		LastRunID:    UUIDToString(row.LastRunID),
		ServedCount:  row.ServedCount,
	}
}

func ToUserVideoServingState(row recommendationsqlc.RecommendationUserVideoServingState) model.UserVideoServingState {
	return model.UserVideoServingState{
		UserID:       UUIDToString(row.UserID),
		VideoID:      UUIDToString(row.VideoID),
		LastServedAt: TimePointerFromPG(row.LastServedAt),
		LastRunID:    UUIDToString(row.LastRunID),
		ServedCount:  row.ServedCount,
	}
}
