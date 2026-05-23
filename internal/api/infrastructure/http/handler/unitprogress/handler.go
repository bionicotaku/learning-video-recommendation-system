package unitprogress

import (
	"context"
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

type ListUserUnitProgressUsecase interface {
	Execute(ctx context.Context, request learningdto.ListUserUnitProgressRequest) (learningdto.ListUserUnitProgressResponse, error)
}

type Handler struct {
	usecase ListUserUnitProgressUsecase
}

func NewHandler(usecase ListUserUnitProgressUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/learning/unit-progress/mastered", h.listMastered)
	mux.HandleFunc("GET /api/learning/unit-progress/unmastered", h.listUnmastered)
}

func writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	httperror.Write(w, r, err, httperror.LearningValidation)
}

func requiredPrincipal(r *http.Request) (auth.Principal, error) {
	return auth.RequirePrincipal(r.Context())
}
