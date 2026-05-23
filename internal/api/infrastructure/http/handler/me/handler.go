package me

import (
	"context"
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
)

type GetMeUsecase interface {
	Execute(ctx context.Context, request userdto.MeRequest) (userdto.MeResponse, error)
}

type UpdateMeProfileUsecase interface {
	Execute(ctx context.Context, request userdto.UpdateMeProfileRequest) (userdto.UpdateMeProfileResponse, error)
}

type Handler struct {
	getMe         GetMeUsecase
	updateProfile UpdateMeProfileUsecase
}

func NewHandler(getMe GetMeUsecase, updateProfile UpdateMeProfileUsecase) *Handler {
	return &Handler{getMe: getMe, updateProfile: updateProfile}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/me", h.handleGetMe)
	mux.HandleFunc("PATCH /api/me/profile", h.handleUpdateProfile)
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

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	httperror.Write(w, r, err,
		httperror.AuthUserNotFound,
		httperror.UserValidation,
	)
}

func requiredPrincipal(r *http.Request) (auth.Principal, error) {
	return auth.RequirePrincipal(r.Context())
}
