package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/dto"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	appservice "learning-video-recommendation-system/internal/recommendation/scheduler/application/service"
)

type ReplayUserUnitStatesUseCase struct {
	txManager apprepo.TxManager
	stateRepo apprepo.UserUnitStateRepository
	eventRepo apprepo.UnitLearningEventRepository
	rebuilder appservice.UserStateRebuilder
}

func NewReplayUserUnitStatesUseCase(
	txManager apprepo.TxManager,
	stateRepo apprepo.UserUnitStateRepository,
	eventRepo apprepo.UnitLearningEventRepository,
	rebuilder appservice.UserStateRebuilder,
) ReplayUserUnitStatesUseCase {
	return ReplayUserUnitStatesUseCase{
		txManager: txManager,
		stateRepo: stateRepo,
		eventRepo: eventRepo,
		rebuilder: rebuilder,
	}
}

func (uc ReplayUserUnitStatesUseCase) Execute(ctx context.Context, cmd command.ReplayStateCommand) (dto.ReplayStateResult, error) {
	result := dto.ReplayStateResult{}

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
		return dto.ReplayStateResult{RebuiltCount: result.RebuiltCount, ErrorCount: 1}, err
	}

	return result, nil
}
