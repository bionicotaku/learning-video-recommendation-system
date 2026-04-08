// 作用：实现事件仓储接口，负责事件表的追加写入和按用户顺序读取。
// 输入/输出：输入是 []LearningEvent 或 userID；输出是 error 或按 occurred_at/event_id 排序的 []LearningEvent。
// 谁调用它：record/replay use case，通过 application/repository/UnitLearningEventRepository 接口调用；fixture 负责装配。
// 它调用谁/传给谁：调用 querier_resolver.go、unit_learning_event_mapper.go 和 sqlcgen/unit_events.sql.go；读取结果会传回 use case。
package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

type unitLearningEventRepository struct {
	querier sqlcgen.Querier
}

func NewUnitLearningEventRepository(querier sqlcgen.Querier) apprepo.UnitLearningEventRepository {
	return unitLearningEventRepository{querier: querier}
}

func (r unitLearningEventRepository) Append(ctx context.Context, events []model.LearningEvent) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	for _, event := range events {
		params, err := mapper.LearningEventToInsertParams(event)
		if err != nil {
			return err
		}
		if err := q.InsertUnitLearningEvent(ctx, params); err != nil {
			return err
		}
	}

	return nil
}

func (r unitLearningEventRepository) ListByUserOrdered(ctx context.Context, userID uuid.UUID) ([]model.LearningEvent, error) {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return nil, err
	}

	rows, err := q.ListUnitLearningEventsByUserOrdered(ctx, mapper.UUIDToPG(userID))
	if err != nil {
		return nil, err
	}

	items := make([]model.LearningEvent, 0, len(rows))
	for _, row := range rows {
		item, err := mapper.LearningEventFromRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}
