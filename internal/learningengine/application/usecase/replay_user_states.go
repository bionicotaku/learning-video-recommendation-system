package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/application/command"
	"learning-video-recommendation-system/internal/learningengine/application/dto"
	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	appservice "learning-video-recommendation-system/internal/learningengine/application/service"
)

type ReplayUserStatesUseCase struct {
	txManager apprepo.TxManager
	stateRepo apprepo.UserUnitStateRepository
	eventRepo apprepo.UnitLearningEventRepository
	rebuilder appservice.UserStateRebuilder
}

func NewReplayUserStatesUseCase(
	txManager apprepo.TxManager,
	stateRepo apprepo.UserUnitStateRepository,
	eventRepo apprepo.UnitLearningEventRepository,
	rebuilder appservice.UserStateRebuilder,
) ReplayUserStatesUseCase {
	return ReplayUserStatesUseCase{
		txManager: txManager,
		stateRepo: stateRepo,
		eventRepo: eventRepo,
		rebuilder: rebuilder,
	}
}

func (uc ReplayUserStatesUseCase) Execute(ctx context.Context, cmd command.ReplayUserStatesCommand) (dto.ReplayUserStatesResult, error) {
	result := dto.ReplayUserStatesResult{}

	err := uc.txManager.WithinTx(ctx, func(ctx context.Context) error {
		events, err := uc.eventRepo.ListByUserOrdered(ctx, cmd.UserID)
		if err != nil {
			return err
		}

		if err := uc.stateRepo.DeleteByUser(ctx, cmd.UserID); err != nil {
			return err
		}

		states, err := uc.rebuilder.Rebuild(events)
		if err != nil {
			return err
		}
		if len(states) == 0 {
			result.RebuiltCount = 0
			return nil
		}

		if err := uc.stateRepo.BatchUpsert(ctx, states); err != nil {
			return err
		}

		result.RebuiltCount = len(states)
		return nil
	})
	if err != nil {
		return dto.ReplayUserStatesResult{RebuiltCount: result.RebuiltCount, ErrorCount: 1}, err
	}

	return result, nil
}
