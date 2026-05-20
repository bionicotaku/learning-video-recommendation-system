package feed

import (
	"encoding/json"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

const (
	defaultTargetVideoCount = 8
	maxTargetVideoCount     = 20
)

type feedBody struct {
	TargetVideoCount int             `json:"target_video_count"`
	ClientContext    json.RawMessage `json:"client_context"`
}

func (h *Handler) getFeed(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := validateContentType(r); err != nil {
		writeHandlerError(w, r, err)
		return
	}

	var body feedBody
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	command, err := mapFeedBody(principal.UserID, body)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	result, err := h.service.Execute(r.Context(), command)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func mapFeedBody(userID string, body feedBody) (apvdto.GetFeedRequest, error) {
	targetCount := body.TargetVideoCount
	if targetCount == 0 {
		targetCount = defaultTargetVideoCount
	}
	if targetCount < 0 || targetCount > maxTargetVideoCount {
		return apvdto.GetFeedRequest{}, apiservice.InvalidRequestError("target_video_count must be between 1 and 20")
	}

	if len(body.ClientContext) == 0 {
		body.ClientContext = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("client_context", body.ClientContext); err != nil {
		return apvdto.GetFeedRequest{}, invalidRequest(err)
	}

	return apvdto.GetFeedRequest{
		UserID:           userID,
		TargetVideoCount: targetCount,
		ClientContext:    body.ClientContext,
	}, nil
}
