package repository

import (
	"context"

	learningrepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
)

type ActivateCollectionRepositories interface {
	TargetCommands() learningrepo.TargetStateCommandRepository
	UserProfiles() userrepo.ProfileRepository
}

type ActivateCollectionTxManager interface {
	WithinUserTx(ctx context.Context, userID string, fn func(ctx context.Context, repos ActivateCollectionRepositories) error) error
}
