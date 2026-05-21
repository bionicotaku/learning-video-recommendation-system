package unitcollections

import (
	"net/http"
	"regexp"
	"strings"

	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
)

var collectionSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,80}$`)

type activateUnitCollectionRequest struct {
	CollectionSlug string `json:"collection_slug"`
}

func (h *Handler) activateUnitCollection(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	var payload activateUnitCollectionRequest
	if err := request.DecodeJSONObject(r.Body, &payload); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}
	collectionSlug := strings.TrimSpace(payload.CollectionSlug)
	if !collectionSlugPattern.MatchString(collectionSlug) {
		writeHandlerError(w, r, invalidRequestText("collection_slug is invalid"))
		return
	}

	result, err := h.activateTarget.Execute(r.Context(), learningdto.ActivateUnitCollectionTargetRequest{
		UserID:         principal.UserID,
		CollectionSlug: collectionSlug,
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if h.updateOnboarding != nil {
		if _, err := h.updateOnboarding.Execute(r.Context(), userdto.UpdateOnboardingStatusRequest{
			UserID: principal.UserID,
			Status: "collection_selected",
		}); err != nil {
			writeHandlerError(w, r, err)
			return
		}
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func invalidRequestText(message string) error {
	return invalidRequest(simpleError(message))
}

type simpleError string

func (e simpleError) Error() string {
	return string(e)
}
