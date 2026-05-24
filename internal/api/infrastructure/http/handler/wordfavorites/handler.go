package wordfavorites

import (
	"context"
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

type GetWordFavoriteStatusUsecase interface {
	Execute(ctx context.Context, request catalogdto.GetWordFavoriteStatusRequest) (catalogdto.WordFavoriteStatusResponse, error)
}

type SetWordFavoriteUsecase interface {
	Execute(ctx context.Context, request catalogdto.SetWordFavoriteRequest) error
}

type UnsetWordFavoriteUsecase interface {
	Execute(ctx context.Context, request catalogdto.UnsetWordFavoriteRequest) error
}

type ListWordFavoritesUsecase interface {
	Execute(ctx context.Context, request catalogdto.ListWordFavoritesRequest) (catalogdto.WordFavoriteListPage, error)
}

type Handler struct {
	status GetWordFavoriteStatusUsecase
	set    SetWordFavoriteUsecase
	unset  UnsetWordFavoriteUsecase
	list   ListWordFavoritesUsecase
}

func NewHandler(status GetWordFavoriteStatusUsecase, set SetWordFavoriteUsecase, unset UnsetWordFavoriteUsecase, list ListWordFavoritesUsecase) *Handler {
	return &Handler{status: status, set: set, unset: unset, list: list}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/word-favorites/status", h.getStatus)
	mux.HandleFunc("PUT /api/word-favorites", h.setFavorite)
	mux.HandleFunc("DELETE /api/word-favorites", h.unsetFavorite)
	mux.HandleFunc("GET /api/word-favorites", h.listFavorites)
}

func requiredPrincipal(r *http.Request) (auth.Principal, error) {
	return auth.RequirePrincipal(r.Context())
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	httperror.Write(w, r, err,
		httperror.CatalogValidation,
		httperror.CatalogNotFound,
	)
}

func invalidRequest(err error) error {
	if err == nil {
		return nil
	}
	return httperror.InvalidRequest(err)
}
