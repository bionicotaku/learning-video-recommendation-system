package videointeractions

import (
	"context"
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

type SetVideoLikeUsecase interface {
	Execute(ctx context.Context, request catalogdto.SetVideoLikeRequest) (catalogdto.VideoLikeResponse, error)
}

type SetVideoFavoriteUsecase interface {
	Execute(ctx context.Context, request catalogdto.SetVideoFavoriteRequest) (catalogdto.VideoFavoriteResponse, error)
}

type Handler struct {
	setLike     SetVideoLikeUsecase
	setFavorite SetVideoFavoriteUsecase
}

func NewHandler(setLike SetVideoLikeUsecase, setFavorite SetVideoFavoriteUsecase) *Handler {
	return &Handler{setLike: setLike, setFavorite: setFavorite}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/videos/{video_id}/like", h.setVideoLike)
	mux.HandleFunc("DELETE /api/videos/{video_id}/like", h.unsetVideoLike)
	mux.HandleFunc("PUT /api/videos/{video_id}/favorite", h.setVideoFavorite)
	mux.HandleFunc("DELETE /api/videos/{video_id}/favorite", h.unsetVideoFavorite)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	httperror.Write(w, r, err,
		httperror.CatalogValidation,
		httperror.CatalogNotFound,
		httperror.CatalogConflict,
		httperror.CatalogUnprocessable,
	)
}

func requiredPrincipal(r *http.Request) (auth.Principal, error) {
	return auth.RequirePrincipal(r.Context())
}

func invalidRequest(err error) error {
	if err == nil {
		return nil
	}
	return httperror.InvalidRequest(err)
}
