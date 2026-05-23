package videodetail

import (
	"context"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
)

type VideoDetailService interface {
	Execute(ctx context.Context, request apvdto.GetVideoDetailRequest) (apvdto.VideoDetailResponse, error)
}

type Handler struct {
	service VideoDetailService
}

func NewHandler(service VideoDetailService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/videos/{video_id}", h.getVideoDetail)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	httperror.Write(w, r, err,
		httperror.CatalogValidation,
		httperror.CatalogNotFound,
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
