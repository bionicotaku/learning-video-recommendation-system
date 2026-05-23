package unitcollections

import (
	"context"
	"net/http"

	apivdto "learning-video-recommendation-system/internal/api/application/dto"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
)

type ListUnitCollectionsUsecase interface {
	Execute(ctx context.Context, request apivdto.ListUnitCollectionsRequest) (apivdto.UnitCollectionsResponse, error)
}

type Handler struct {
	listCollections ListUnitCollectionsUsecase
}

func NewHandler(listCollections ListUnitCollectionsUsecase) *Handler {
	return &Handler{listCollections: listCollections}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/unit-collections", h.listUnitCollections)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	httperror.Write(w, r, err,
		httperror.LearningValidation,
		httperror.UserValidation,
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
