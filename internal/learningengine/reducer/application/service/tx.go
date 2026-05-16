package service

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
)

type TransactionalRepositories interface {
	UserUnitStates() apprepo.UserUnitStateRepository
	TargetCommands() apprepo.TargetStateCommandRepository
	UnitLearningEvents() apprepo.UnitLearningEventRepository
}

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, repos TransactionalRepositories) error) error
	WithinUserTx(ctx context.Context, userID string, fn func(ctx context.Context, repos TransactionalRepositories) error) error
}
