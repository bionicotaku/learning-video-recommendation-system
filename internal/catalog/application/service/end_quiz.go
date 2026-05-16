package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	"learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

const maxEndQuizUnitCount = 8

type EndQuizQuestionLookupUsecase struct {
	reader repository.EndQuizQuestionReader
}

func NewEndQuizQuestionLookupUsecase(reader repository.EndQuizQuestionReader) *EndQuizQuestionLookupUsecase {
	return &EndQuizQuestionLookupUsecase{reader: reader}
}

func (u *EndQuizQuestionLookupUsecase) Execute(ctx context.Context, request dto.EndQuizQuestionLookupRequest) (dto.EndQuizQuestionLookupResponse, error) {
	if u.reader == nil {
		return dto.EndQuizQuestionLookupResponse{}, errors.New("end quiz question reader is required")
	}
	videoID := strings.TrimSpace(request.VideoID)
	if videoID == "" {
		return dto.EndQuizQuestionLookupResponse{}, validationError("video_id is required")
	}
	unitIDs, err := normalizeEndQuizUnitIDs(request.CoarseUnitIDs)
	if err != nil {
		return dto.EndQuizQuestionLookupResponse{}, err
	}

	visible, err := u.reader.HasVisibleVideoForEndQuiz(ctx, videoID)
	if err != nil {
		return dto.EndQuizQuestionLookupResponse{}, err
	}
	if !visible {
		return dto.EndQuizQuestionLookupResponse{}, NotFoundError("video is not available for end quiz")
	}

	videoCandidates, err := u.reader.ListVideoUnitQuizQuestionCandidates(ctx, videoID, unitIDs)
	if err != nil {
		return dto.EndQuizQuestionLookupResponse{}, err
	}
	unitCandidates, err := u.reader.ListUnitQuizQuestionCandidates(ctx, unitIDs)
	if err != nil {
		return dto.EndQuizQuestionLookupResponse{}, err
	}

	videoByUnit := groupEndQuizCandidates(videoCandidates)
	unitByUnit := groupEndQuizCandidates(unitCandidates)
	items := make([]dto.EndQuizItem, 0, len(unitIDs))
	missing := make([]int64, 0)
	for _, unitID := range unitIDs {
		item, ok := firstValidEndQuizItem(videoByUnit[unitID], "video_context")
		if !ok {
			item, ok = firstValidEndQuizItem(unitByUnit[unitID], "unit_generic")
		}
		if !ok {
			missing = append(missing, unitID)
			continue
		}
		items = append(items, item)
	}

	return dto.EndQuizQuestionLookupResponse{
		VideoID:              videoID,
		Items:                items,
		MissingCoarseUnitIDs: missing,
	}, nil
}

func normalizeEndQuizUnitIDs(values []int64) ([]int64, error) {
	if len(values) == 0 {
		return nil, validationError("coarse_unit_ids is required")
	}
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			return nil, validationError("coarse_unit_ids must contain positive integers")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, validationError("coarse_unit_ids is required")
	}
	if len(result) > maxEndQuizUnitCount {
		return nil, validationError("coarse_unit_ids must contain at most 8 items")
	}
	return result, nil
}

func groupEndQuizCandidates(candidates []model.EndQuizQuestionCandidate) map[int64][]model.EndQuizQuestionCandidate {
	result := make(map[int64][]model.EndQuizQuestionCandidate)
	for _, candidate := range candidates {
		result[candidate.CoarseUnitID] = append(result[candidate.CoarseUnitID], candidate)
	}
	return result
}

func firstValidEndQuizItem(candidates []model.EndQuizQuestionCandidate, source string) (dto.EndQuizItem, bool) {
	for _, candidate := range candidates {
		item, err := mapEndQuizCandidate(candidate, source)
		if err == nil {
			return item, true
		}
	}
	return dto.EndQuizItem{}, false
}

type endQuizPayload struct {
	Question    string                 `json:"question"`
	ContextText *string                `json:"context_text"`
	Options     []endQuizPayloadOption `json:"options"`
	Explanation *string                `json:"explanation"`
}

type endQuizPayloadOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func mapEndQuizCandidate(candidate model.EndQuizQuestionCandidate, source string) (dto.EndQuizItem, error) {
	var payload endQuizPayload
	if err := json.Unmarshal(candidate.ContentPayload, &payload); err != nil {
		return dto.EndQuizItem{}, err
	}
	if strings.TrimSpace(payload.Question) == "" {
		return dto.EndQuizItem{}, errors.New("question is required")
	}
	if len(payload.Options) == 0 {
		return dto.EndQuizItem{}, errors.New("options is required")
	}
	options := make([]dto.EndQuizOption, 0, len(payload.Options))
	hasCorrect := false
	for _, option := range payload.Options {
		if strings.TrimSpace(option.ID) == "" || strings.TrimSpace(option.Text) == "" {
			return dto.EndQuizItem{}, errors.New("option id and text are required")
		}
		if option.ID == "correct" {
			hasCorrect = true
		}
		options = append(options, dto.EndQuizOption{
			OptionID: option.ID,
			Text:     option.Text,
		})
	}
	if !hasCorrect {
		return dto.EndQuizItem{}, errors.New("correct option is required")
	}

	return dto.EndQuizItem{
		CoarseUnitID:         candidate.CoarseUnitID,
		QuestionID:           candidate.QuestionID,
		Source:               source,
		QuestionType:         candidate.QuestionType,
		TargetText:           candidate.TargetText,
		Question:             payload.Question,
		ContextText:          nonBlankStringPointer(payload.ContextText),
		Options:              options,
		Explanation:          nonBlankStringPointer(payload.Explanation),
		ContextSentenceIndex: candidate.ContextSentenceIndex,
		ContextSpanIndex:     candidate.ContextSpanIndex,
		ContextStartMS:       candidate.ContextStartMS,
		ContextEndMS:         candidate.ContextEndMS,
	}, nil
}

func nonBlankStringPointer(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return value
}
