// 文件作用：
//   - 实现 LearningStateSnapshotReadRepository
//   - 负责执行 candidate 查询并把结果转换成 application/query 层候选对象
//
// 输入/输出：
//   - 输入：userID、now
//   - 输出：[]ReviewCandidate 或 []NewCandidate
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//   - 集成测试 candidate_queries_test.go 会直接验证它
//
// 它调用谁/传给谁：
//   - 调用 resolveQuerier 选择 tx 或普通 querier
//   - 调用 sqlcgen.FindDueReviewCandidates / FindNewCandidates
//   - 调用 mapper.ReviewCandidatesFromRows / NewCandidatesFromRows
//   - 把候选结果传给 usecase
package repository

import (
	"context"
	"time"

	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

type userUnitStateReadRepository struct {
	querier sqlcgen.Querier
}

func NewLearningStateSnapshotReadRepository(querier sqlcgen.Querier) apprepo.LearningStateSnapshotReadRepository {
	return userUnitStateReadRepository{querier: querier}
}

func (r userUnitStateReadRepository) FindDueReviewCandidates(ctx context.Context, userID uuid.UUID, now time.Time) ([]appquery.ReviewCandidate, error) {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return nil, err
	}

	rows, err := q.FindDueReviewCandidates(ctx, sqlcgen.FindDueReviewCandidatesParams{
		UserID: mapper.UUIDToPG(userID),
		Now:    mapper.TimeToPG(now),
	})
	if err != nil {
		return nil, err
	}

	return mapper.ReviewCandidatesFromRows(rows)
}

func (r userUnitStateReadRepository) FindNewCandidates(ctx context.Context, userID uuid.UUID) ([]appquery.NewCandidate, error) {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return nil, err
	}

	rows, err := q.FindNewCandidates(ctx, mapper.UUIDToPG(userID))
	if err != nil {
		return nil, err
	}

	return mapper.NewCandidatesFromRows(rows)
}
