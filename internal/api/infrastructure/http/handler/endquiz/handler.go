package endquiz

import (
	"context"
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

type EndQuizQuestionLookupUsecase interface {
	Execute(ctx context.Context, request catalogdto.EndQuizQuestionLookupRequest) (catalogdto.EndQuizQuestionLookupResponse, error)
}

type Handler struct {
	lookup EndQuizQuestionLookupUsecase
}

func NewHandler(lookup EndQuizQuestionLookupUsecase) *Handler {
	return &Handler{lookup: lookup}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/videos/end-quiz", h.getEndQuiz)
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
