// 文件作用：
//   - 实现 UserUnitServingStateRepository
//   - 负责触碰本轮被推荐 coarse unit 的最近推荐时间和最近 run ID
//
// 输入/输出：
//   - 输入：userID、runID、coarseUnitIDs、recommendedAt
//   - 输出：serving state 更新成功或失败
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//   - 集成测试 usecase 场景会间接覆盖它
//
// 它调用谁/传给谁：
//   - 调用 resolveQuerier
//   - 调用 uniqueInt64s 去重
//   - 调用 mapper.UserUnitServingStateToUpsertParams
//   - 调用 sqlcgen.UpsertUserUnitServingState
package repository

import (
	"context"
	"time"

	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

type userUnitServingStateRepository struct {
	querier sqlcgen.Querier
}

func NewUserUnitServingStateRepository(querier sqlcgen.Querier) apprepo.UserUnitServingStateRepository {
	return userUnitServingStateRepository{querier: querier}
}

func (r userUnitServingStateRepository) TouchRecommendedAt(ctx context.Context, userID uuid.UUID, runID uuid.UUID, coarseUnitIDs []int64, recommendedAt time.Time) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	for _, coarseUnitID := range uniqueInt64s(coarseUnitIDs) {
		if err := q.UpsertUserUnitServingState(ctx, mapper.UserUnitServingStateToUpsertParams(userID, coarseUnitID, runID, recommendedAt)); err != nil {
			return err
		}
	}

	return nil
}

func uniqueInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
