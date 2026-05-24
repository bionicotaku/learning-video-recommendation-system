package wordfavorites

import (
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
)

func (h *Handler) setFavorite(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := request.RequireJSONContentType(r); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}
	var body wordFavoriteIdentityRequest
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}
	occurredAt, err := request.ParseRequiredTime("occurred_at", body.OccurredAt)
	if err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}
	if err := h.set.Execute(r.Context(), body.setDTO(principal.UserID, occurredAt)); err != nil {
		writeHandlerError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) unsetFavorite(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := request.RequireJSONContentType(r); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}
	var body wordFavoriteIdentityRequest
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}
	occurredAt, err := request.ParseRequiredTime("occurred_at", body.OccurredAt)
	if err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}
	if err := h.unset.Execute(r.Context(), body.unsetDTO(principal.UserID, occurredAt)); err != nil {
		writeHandlerError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
