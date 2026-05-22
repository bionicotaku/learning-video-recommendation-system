package main

import (
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/feedback"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/me"
	userservice "learning-video-recommendation-system/internal/user/application/service"
	userrepo "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func buildMeHandler(pool *pgxpool.Pool) *me.Handler {
	repository := userrepo.NewRepository(pool)
	getMe := userservice.NewGetMeUsecase(repository, repository)
	return me.NewHandler(getMe)
}

func buildFeedbackHandler(pool *pgxpool.Pool) *feedback.Handler {
	writer := userrepo.NewFeedbackWriter(pool)
	submitFeedback := userservice.NewSubmitFeedbackUsecase(writer)
	return feedback.NewHandler(submitFeedback)
}
