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
	defaultMinDurationSec   = 45
	defaultMaxDurationSec   = 180
)

type feedBody struct {
	TargetVideoCount     int                    `json:"target_video_count"`
	PreferredDurationSec *preferredDurationBody `json:"preferred_duration_sec"`
	SessionHint          string                 `json:"session_hint"`
	ClientContext        json.RawMessage        `json:"client_context"`
}

type preferredDurationBody struct {
	Min *int `json:"min"`
	Max *int `json:"max"`
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

	preferredDuration := [2]int{defaultMinDurationSec, defaultMaxDurationSec}
	if body.PreferredDurationSec != nil {
		if body.PreferredDurationSec.Min != nil {
			if *body.PreferredDurationSec.Min <= 0 {
				return apvdto.GetFeedRequest{}, apiservice.InvalidRequestError("preferred_duration_sec.min must be positive")
			}
			preferredDuration[0] = *body.PreferredDurationSec.Min
		}
		if body.PreferredDurationSec.Max != nil {
			if *body.PreferredDurationSec.Max <= 0 {
				return apvdto.GetFeedRequest{}, apiservice.InvalidRequestError("preferred_duration_sec.max must be positive")
			}
			preferredDuration[1] = *body.PreferredDurationSec.Max
		}
		if preferredDuration[1] < preferredDuration[0] {
			return apvdto.GetFeedRequest{}, apiservice.InvalidRequestError("preferred_duration_sec.max must be greater than or equal to min")
		}
	}

	if len(body.ClientContext) == 0 {
		body.ClientContext = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("client_context", body.ClientContext); err != nil {
		return apvdto.GetFeedRequest{}, invalidRequest(err)
	}

	return apvdto.GetFeedRequest{
		UserID:               userID,
		TargetVideoCount:     targetCount,
		PreferredDurationSec: preferredDuration,
		SessionHint:          body.SessionHint,
		ClientContext:        body.ClientContext,
	}, nil
}
