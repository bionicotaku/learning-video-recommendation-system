package videolibrary

import (
	"context"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
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
	httperror.Write(w, r, err, httperror.CatalogValidation)
}

func requiredPrincipal(r *http.Request) (auth.Principal, error) {
	return auth.RequirePrincipal(r.Context())
}
