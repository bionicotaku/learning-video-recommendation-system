// 文件作用：
//   - 实现 SchedulerRunRepository
//   - 负责把 RecommendationBatch 拆成 run 头和 run items，并落到 recommendation.scheduler_runs / scheduler_run_items
//
// 输入/输出：
//   - 输入：RecommendationBatch
//   - 输出：落库成功或失败
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//   - 集成测试 usecase 场景会间接覆盖它
//
// 它调用谁/传给谁：
//   - 调用 resolveQuerier
//   - 调用 mapper.SchedulerRunParamsFromBatch / SchedulerRunItemParamsFromBatch
//   - 调用 sqlcgen.UpsertSchedulerRun / UpsertSchedulerRunItem
package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

type schedulerRunRepository struct {
	querier sqlcgen.Querier
}

func NewSchedulerRunRepository(querier sqlcgen.Querier) apprepo.SchedulerRunRepository {
	return schedulerRunRepository{querier: querier}
}

func (r schedulerRunRepository) SaveRun(ctx context.Context, batch model.RecommendationBatch) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	params, err := mapper.SchedulerRunParamsFromBatch(batch)
	if err != nil {
		return err
	}

	return q.UpsertSchedulerRun(ctx, params)
}

func (r schedulerRunRepository) SaveRunItems(ctx context.Context, batch model.RecommendationBatch) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	params, err := mapper.SchedulerRunItemParamsFromBatch(batch)
	if err != nil {
		return err
	}

	for _, item := range params {
		if err := q.UpsertSchedulerRunItem(ctx, item); err != nil {
			return err
		}
	}

	return nil
}
