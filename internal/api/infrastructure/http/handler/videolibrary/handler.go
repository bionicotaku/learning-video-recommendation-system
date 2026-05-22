package videolibrary

import (
	"context"
	"errors"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
)

type VideoLibraryService interface {
	ListFavorites(ctx context.Context, request apvdto.ListVideoFavoritesRequest) (apvdto.ListVideoFavoritesResponse, error)
	ListHistory(ctx context.Context, request apvdto.ListVideoHistoryRequest) (apvdto.ListVideoHistoryResponse, error)
}

type Handler struct {
	service VideoLibraryService
}

func NewHandler(service VideoLibraryService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/video-favorites", h.listVideoFavorites)
	mux.HandleFunc("GET /api/video-history", h.listVideoHistory)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	switch {
	case errors.Is(err, auth.ErrMissingPrincipal):
		response.WriteError(w, requestID, response.Unauthorized("trusted principal is required"))
	case apiservice.IsInvalidRequest(err), catalogservice.IsValidationError(err):
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
