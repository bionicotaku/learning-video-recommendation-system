package feed

import (
	"context"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
)

type FeedService interface {
	Execute(ctx context.Context, request apvdto.GetFeedRequest) (apvdto.FeedResponse, error)
}

type Handler struct {
	service FeedService
}

func NewHandler(service FeedService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/feed", h.getFeed)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	httperror.Write(w, r, err)
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
