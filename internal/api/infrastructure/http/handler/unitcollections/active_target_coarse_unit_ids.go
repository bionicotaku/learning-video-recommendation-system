package unitcollections

import (
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

func (h *Handler) getActiveLearningTargetCoarseUnitIDs(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	result, err := h.activeTargetUnitIDs.Execute(r.Context(), learningdto.GetActiveLearningTargetCoarseUnitIDsRequest{
		UserID: principal.UserID,
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}
