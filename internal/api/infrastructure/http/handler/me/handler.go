package me

import (
	"context"
	"errors"
	"net/http"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	userservice "learning-video-recommendation-system/internal/user/application/service"
)

type GetMeUsecase interface {
	Execute(ctx context.Context, request userdto.MeRequest) (userdto.MeResponse, error)
}

type GetActivityCalendarUsecase interface {
	Execute(ctx context.Context, request userdto.ActivityCalendarRequest) (userdto.ActivityCalendarResponse, error)
}

type Handler struct {
	getMe               GetMeUsecase
	getActivityCalendar GetActivityCalendarUsecase
}

func NewHandler(getMe GetMeUsecase, getActivityCalendar GetActivityCalendarUsecase) *Handler {
	return &Handler{getMe: getMe, getActivityCalendar: getActivityCalendar}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/me", h.handleGetMe)
	mux.HandleFunc("GET /api/me/activity-calendar", h.handleGetActivityCalendar)
}

func (h *Handler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	result, err := h.getMe.Execute(r.Context(), userdto.MeRequest{
		UserID:         principal.UserID,
		ClientTimezone: r.Header.Get("X-Client-Timezone"),
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) handleGetActivityCalendar(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	result, err := h.getActivityCalendar.Execute(r.Context(), userdto.ActivityCalendarRequest{
		UserID:         principal.UserID,
		ClientTimezone: r.Header.Get("X-Client-Timezone"),
	})
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	switch {
	case errors.Is(err, auth.ErrMissingPrincipal), errors.Is(err, userrepo.ErrAuthUserNotFound):
		response.WriteError(w, requestID, response.Unauthorized("trusted principal is required"))
	case apiservice.IsInvalidRequest(err), userservice.IsValidationError(err):
		response.WriteError(w, requestID, response.InvalidRequest(err.Error()))
	case apiservice.IsServiceUnavailable(err), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		response.WriteError(w, requestID, response.ServiceUnavailable("request canceled or timed out"))
	default:
		response.WriteError(w, requestID, response.InternalError())
	}
}

func requiredPrincipal(r *http.Request) (auth.Principal, error) {
	return auth.RequirePrincipal(r.Context())
}
