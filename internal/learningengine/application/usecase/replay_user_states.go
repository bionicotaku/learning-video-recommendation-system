// 作用：实现 full replay 主用例，负责读取事件历史、清空旧状态并重建用户全部状态。
// 输入/输出：输入是 ReplayUserStatesCommand；输出是 ReplayUserStatesResult 或 error。
// 谁调用它：上层业务调用方、integration/usecase/replay_user_states_usecase_test.go、fixture/helpers.go。
// 它调用谁/传给谁：调用 TxManager、两个 repository 和 UserStateRebuilder；最终把 replay 结果返回给调用方。
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
