package endquiz

import (
	"encoding/json"
	"net/http"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

const maxEndQuizUnitCount = 8

type endQuizBody struct {
	VideoID             string          `json:"video_id"`
	CoarseUnitIDs       []int64         `json:"coarse_unit_ids"`
	RecommendationRunID string          `json:"recommendation_run_id"`
	ClientContext       json.RawMessage `json:"client_context"`
}

func (h *Handler) getEndQuiz(w http.ResponseWriter, r *http.Request) {
	if _, err := requiredPrincipal(r); err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := validateContentType(r); err != nil {
		writeHandlerError(w, r, err)
		return
	}

	var body endQuizBody
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	command, err := mapEndQuizBody(body)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	result, err := h.lookup.Execute(r.Context(), command)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func mapEndQuizBody(body endQuizBody) (catalogdto.EndQuizQuestionLookupRequest, error) {
	if err := request.ValidateRequiredUUID("video_id", body.VideoID); err != nil {
		return catalogdto.EndQuizQuestionLookupRequest{}, invalidRequest(err)
	}
	if err := request.ValidateOptionalUUID("recommendation_run_id", body.RecommendationRunID); err != nil {
		return catalogdto.EndQuizQuestionLookupRequest{}, invalidRequest(err)
	}
	unitIDs, err := normalizeCoarseUnitIDs(body.CoarseUnitIDs)
	if err != nil {
		return catalogdto.EndQuizQuestionLookupRequest{}, err
	}
	if len(body.ClientContext) == 0 {
		body.ClientContext = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("client_context", body.ClientContext); err != nil {
		return catalogdto.EndQuizQuestionLookupRequest{}, invalidRequest(err)
	}

	return catalogdto.EndQuizQuestionLookupRequest{
		VideoID:       body.VideoID,
		CoarseUnitIDs: unitIDs,
	}, nil
}

func normalizeCoarseUnitIDs(values []int64) ([]int64, error) {
	if len(values) == 0 {
		return nil, apiservice.InvalidRequestError("coarse_unit_ids is required")
	}
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			return nil, apiservice.InvalidRequestError("coarse_unit_ids must contain positive integers")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) > maxEndQuizUnitCount {
		return nil, apiservice.InvalidRequestError("coarse_unit_ids must contain at most 8 items")
	}
	return result, nil
}
