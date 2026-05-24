package mapper

import (
	"encoding/json"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

func LearningUnitsToJSON(units []model.ExpectedLearningUnit) ([]byte, error) {
	if units == nil {
		units = []model.ExpectedLearningUnit{}
	}
	return json.Marshal(units)
}

type recommendationItemJSON struct {
	RunID          string                       `json:"run_id"`
	Rank           int32                        `json:"rank"`
	VideoID        string                       `json:"video_id"`
	Score          float64                      `json:"score"`
	PrimaryLane    string                       `json:"primary_lane"`
	DominantRole   string                       `json:"dominant_role"`
	DominantUnitID *int64                       `json:"dominant_unit_id"`
	ReasonCodes    []string                     `json:"reason_codes"`
	LearningUnits  []model.ExpectedLearningUnit `json:"learning_units"`
}

func RecommendationItemsToJSON(items []model.RecommendationItem) ([]byte, error) {
	if items == nil {
		items = []model.RecommendationItem{}
	}
	payload := make([]recommendationItemJSON, 0, len(items))
	for _, item := range items {
		reasonCodes := item.ReasonCodes
		if reasonCodes == nil {
			reasonCodes = []string{}
		}
		learningUnits := item.LearningUnits
		if learningUnits == nil {
			learningUnits = []model.ExpectedLearningUnit{}
		}
		payload = append(payload, recommendationItemJSON{
			RunID:          item.RunID,
			Rank:           item.Rank,
			VideoID:        item.VideoID,
			Score:          item.Score,
			PrimaryLane:    item.PrimaryLane,
			DominantRole:   string(item.DominantRole),
			DominantUnitID: item.DominantUnitID,
			ReasonCodes:    reasonCodes,
			LearningUnits:  learningUnits,
		})
	}
	return json.Marshal(payload)
}

func ToLearningStateSnapshot(row recommendationsqlc.LearningUserUnitState) (model.LearningStateSnapshot, error) {
	targetPriority, err := NumericToFloat64(row.TargetPriority)
	if err != nil {
		return model.LearningStateSnapshot{}, err
	}
	masteryScore, err := NumericToFloat64(row.MasteryScore)
	if err != nil {
		return model.LearningStateSnapshot{}, err
	}

	return model.LearningStateSnapshot{
		UserID:              UUIDToString(row.UserID),
		CoarseUnitID:        row.CoarseUnitID,
		IsTarget:            row.IsTarget,
		TargetPriority:      targetPriority,
		Status:              row.Status,
		MasteryScore:        masteryScore,
		LastProgressQuality: Int16PointerFromPG(row.LastProgressQuality),
		NextReviewAt:        TimePointerFromPG(row.NextReviewAt),
		UpdatedAt:           TimeFromPG(row.UpdatedAt),
	}, nil
}

func ToRecommendableVideoUnit(row recommendationsqlc.ListVideoUnitRecallRowsByUnitIDsRow) (model.RecommendableVideoUnit, error) {
	coverageRatio, err := NumericToFloat64(row.CoverageRatio)
	if err != nil {
		return model.RecommendableVideoUnit{}, err
	}
	mappedSpanRatio, err := NumericToFloat64(row.MappedSpanRatio)
	if err != nil {
		return model.RecommendableVideoUnit{}, err
	}
	contentQualityScore, err := NumericToFloat64(row.ContentQualityScore)
	if err != nil {
		return model.RecommendableVideoUnit{}, err
	}
	bestEvidenceCandidateScore, err := NumericPointerToFloat64(row.BestEvidenceCandidateScore)
	if err != nil {
		return model.RecommendableVideoUnit{}, err
	}

	return model.RecommendableVideoUnit{
		VideoID:         UUIDToString(row.VideoID),
		CoarseUnitID:    row.CoarseUnitID,
		MentionCount:    row.MentionCount,
		SentenceCount:   row.SentenceCount,
		CoverageMs:      row.CoverageMs,
		CoverageRatio:   coverageRatio,
		SentenceIndexes: row.SentenceIndexes,
		BestEvidenceRef: model.EvidenceRef{
			SentenceIndex: row.BestEvidenceSentenceIndex,
			SpanIndex:     row.BestEvidenceSpanIndex,
		},
		BestEvidenceStartMs:        Int32FromPG(row.BestEvidenceStartMs),
		BestEvidenceEndMs:          Int32FromPG(row.BestEvidenceEndMs),
		BestEvidenceCandidateScore: bestEvidenceCandidateScore,
		BestEvidenceTargetText:     TextPointerFromPG(row.BestEvidenceTargetText),
		DurationMs:                 row.DurationMs,
		MappedSpanRatio:            mappedSpanRatio,
		ContentQualityScore:        contentQualityScore,
		RankWithinUnit:             row.RankWithinUnit,
	}, nil
}

func ToVideoFillCandidateFromMasteredTarget(row recommendationsqlc.ListMasteredTargetFillVideoCandidatesRow) (model.VideoFillCandidate, error) {
	maxCoverageRatio, err := NumericToFloat64(row.MaxCoverageRatio)
	if err != nil {
		return model.VideoFillCandidate{}, err
	}
	mappedSpanRatio, err := NumericToFloat64(row.MappedSpanRatio)
	if err != nil {
		return model.VideoFillCandidate{}, err
	}

	return model.VideoFillCandidate{
		VideoID:           UUIDToString(row.VideoID),
		DurationMs:        row.DurationMs,
		MatchedUnitCount:  row.MatchedUnitCount,
		TotalMentionCount: row.TotalMentionCount,
		MaxCoverageRatio:  maxCoverageRatio,
		MappedSpanRatio:   mappedSpanRatio,
		ViewCount:         row.ViewCount,
		LikeCount:         row.LikeCount,
		FavoriteCount:     row.FavoriteCount,
		LastServedAt:      TimePointerFromPG(row.LastServedAt),
		ServedCount:       row.ServedCount,
		LastWatchedAt:     TimePointerFromPG(row.LastWatchedAt),
		WatchCount:        row.WatchCount,
		CompletedCount:    row.CompletedCount,
		MaxPositionMs:     row.MaxPositionMs,
	}, nil
}

func ToVideoFillCandidateFromPopular(row recommendationsqlc.ListPopularFillVideoCandidatesRow) (model.VideoFillCandidate, error) {
	maxCoverageRatio, err := NumericToFloat64(row.MaxCoverageRatio)
	if err != nil {
		return model.VideoFillCandidate{}, err
	}
	mappedSpanRatio, err := NumericToFloat64(row.MappedSpanRatio)
	if err != nil {
		return model.VideoFillCandidate{}, err
	}

	return model.VideoFillCandidate{
		VideoID:           UUIDToString(row.VideoID),
		DurationMs:        row.DurationMs,
		MatchedUnitCount:  row.MatchedUnitCount,
		TotalMentionCount: row.TotalMentionCount,
		MaxCoverageRatio:  maxCoverageRatio,
		MappedSpanRatio:   mappedSpanRatio,
		ViewCount:         row.ViewCount,
		LikeCount:         row.LikeCount,
		FavoriteCount:     row.FavoriteCount,
		LastServedAt:      TimePointerFromPG(row.LastServedAt),
		ServedCount:       row.ServedCount,
		LastWatchedAt:     TimePointerFromPG(row.LastWatchedAt),
		WatchCount:        row.WatchCount,
		CompletedCount:    row.CompletedCount,
		MaxPositionMs:     row.MaxPositionMs,
	}, nil
}

func ToUnitVideoInventory(row recommendationsqlc.RecommendationVUnitVideoInventory) (model.UnitVideoInventory, error) {
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

	return model.UnitVideoInventory{
		CoarseUnitID:       row.CoarseUnitID,
		DistinctVideoCount: row.DistinctVideoCount,
		AvgMentionCount:    avgMentionCount,
		AvgSentenceCount:   avgSentenceCount,
		AvgCoverageMs:      avgCoverageMs,
		AvgCoverageRatio:   avgCoverageRatio,
		StrongVideoCount:   row.StrongVideoCount,
		SupplyGrade:        row.SupplyGrade,
		UpdatedAt:          TimeFromPG(row.UpdatedAt),
	}, nil
}

func ToVideoUserState(row recommendationsqlc.CatalogVideoUserState) (model.VideoUserState, error) {
	return model.VideoUserState{
		UserID:         UUIDToString(row.UserID),
		VideoID:        UUIDToString(row.VideoID),
		LastWatchedAt:  TimePointerFromPG(row.LastWatchedAt),
		WatchCount:     row.WatchCount,
		CompletedCount: row.CompletedCount,
		LastPositionMs: row.LastPositionMs,
		MaxPositionMs:  row.MaxPositionMs,
		TotalWatchMs:   row.TotalWatchMs,
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
