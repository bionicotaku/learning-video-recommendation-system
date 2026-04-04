package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/dto"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"
	domainservice "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
)

type ReplayUserUnitStatesUseCase struct {
	txManager    apprepo.TxManager
	stateRepo    apprepo.UserUnitStateRepository
	eventRepo    apprepo.UnitLearningEventRepository
	stateUpdater domainservice.StateUpdater
}

func NewReplayUserUnitStatesUseCase(
	txManager apprepo.TxManager,
	stateRepo apprepo.UserUnitStateRepository,
	eventRepo apprepo.UnitLearningEventRepository,
	stateUpdater domainservice.StateUpdater,
) ReplayUserUnitStatesUseCase {
	return ReplayUserUnitStatesUseCase{
		txManager:    txManager,
		stateRepo:    stateRepo,
		eventRepo:    eventRepo,
		stateUpdater: stateUpdater,
	}
}

func (uc ReplayUserUnitStatesUseCase) Execute(ctx context.Context, cmd command.ReplayStateCommand) (dto.ReplayStateResult, error) {
	result := dto.ReplayStateResult{}

	err := uc.txManager.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.stateRepo.DeleteForReplay(ctx, cmd.UserID, cmd.CoarseUnitID); err != nil {
			return err
		}

		events, err := uc.eventRepo.FindForReplay(ctx, cmd.UserID, cmd.CoarseUnitID, cmd.FromTime)
		if err != nil {
			return err
		}

		states := make(map[int64]*model.UserUnitState)
		historyByUnit := make(map[int64][]model.LearningEvent)
		for _, event := range events {
			history := historyByUnit[event.CoarseUnitID]
			current := states[event.CoarseUnitID]
			next, _, err := uc.stateUpdater.Apply(current, event, domainservice.UpdateContext{
				SchedulerPolicy:   policy.DefaultSchedulerPolicy(),
				RecentQualities:   recentQualitiesFromEvents(history),
				RecentCorrectness: recentCorrectnessFromEvents(history),
				Now:               event.CreatedAt,
			})
			if err != nil {
				return err
			}

			states[event.CoarseUnitID] = next
			historyByUnit[event.CoarseUnitID] = append(historyByUnit[event.CoarseUnitID], event)
		}

		upserts := make([]*model.UserUnitState, 0, len(states))
		for _, state := range states {
			upserts = append(upserts, state)
		}

		if err := uc.stateRepo.BatchUpsert(ctx, upserts); err != nil {
			return err
		}

		result.RebuiltCount = len(upserts)
		return nil
	})
	if err != nil {
		return dto.ReplayStateResult{RebuiltCount: result.RebuiltCount, ErrorCount: 1}, err
	}

	return result, nil
}
