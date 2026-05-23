package learningevents

import (
	"context"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
)

type RecordLearningInteractionsBatchService interface {
	Execute(ctx context.Context, request apvdto.RecordLearningInteractionsBatchRequest) (apvdto.RecordLearningInteractionsBatchResponse, error)
}

type RecordQuizAttemptService interface {
	Execute(ctx context.Context, request apvdto.RecordQuizAttemptRequest) (apvdto.RecordQuizAttemptResponse, error)
}

type RecordSelfMarkMasteredService interface {
	Execute(ctx context.Context, request apvdto.RecordSelfMarkMasteredRequest) (apvdto.RecordSelfMarkMasteredResponse, error)
}

type ResetUserUnitProgressService interface {
	Execute(ctx context.Context, request apvdto.ResetUserUnitProgressRequest) (apvdto.ResetUserUnitProgressResponse, error)
}

type Handler struct {
	learningInteractions RecordLearningInteractionsBatchService
	quizAttempts         RecordQuizAttemptService
	selfMarkMastered     RecordSelfMarkMasteredService
	resetProgress        ResetUserUnitProgressService
}

func NewHandler(
	learningInteractions RecordLearningInteractionsBatchService,
	quizAttempts RecordQuizAttemptService,
	selfMarkMastered RecordSelfMarkMasteredService,
	resetUserUnitProgress ResetUserUnitProgressService,
) *Handler {
	return &Handler{
		learningInteractions: learningInteractions,
		quizAttempts:         quizAttempts,
		selfMarkMastered:     selfMarkMastered,
		resetProgress:        resetUserUnitProgress,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/learning-interactions:batch", h.recordLearningInteractionsBatch)
	mux.HandleFunc("POST /api/quiz-attempts", h.recordQuizAttempt)
	mux.HandleFunc("POST /api/learning-units:mark-mastered", h.recordSelfMarkMastered)
	mux.HandleFunc("POST /api/learning-units:reset-unlearned", h.resetUserUnitProgress)
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

func validateOptionalUUIDs(values map[string]string) error {
	for field, value := range values {
		if err := request.ValidateOptionalUUID(field, value); err != nil {
			return err
		}
	}
	return nil
}
