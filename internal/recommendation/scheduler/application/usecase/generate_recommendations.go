// 文件作用：
//   - 实现 scheduler 当前唯一主用例 GenerateLearningUnitRecommendationsUseCase
//   - 负责把候选读取、规则计算、批次组装和 Recommendation 自有表写入串成一条完整链路
//
// 输入/输出：
//   - 输入：GenerateRecommendationsCommand
//   - 输出：GenerateRecommendationsResult，其中包含 RecommendationBatch
//
// 谁调用它：
//   - 外层业务组装代码
//   - 测试夹具 fixture.NewGenerateUseCase 组装后的调用方
//   - 集成测试与场景测试直接调用 Execute
//
// 它调用谁/传给谁：
//   - 调用 LearningStateSnapshotReadRepository 读取候选
//   - 调用 BacklogCalculator / QuotaAllocator / ReviewScorer / NewScorer / PriorityZeroExtractor / RecommendationAssembler
//   - 调用 TxManager 在事务中写 SchedulerRunRepository 和 UserUnitServingStateRepository
//   - 最终把结果返回给上层调用方
package usecase

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/dto"
	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	domainservice "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
)

type GenerateLearningUnitRecommendationsUseCase struct {
	txManager         apprepo.TxManager
	stateRepo         apprepo.LearningStateSnapshotReadRepository
	servingStateRepo  apprepo.UserUnitServingStateRepository
	runRepo           apprepo.SchedulerRunRepository
	backlogCalculator domainservice.BacklogCalculator
	quotaAllocator    domainservice.QuotaAllocator
	reviewScorer      domainservice.ReviewScorer
	newScorer         domainservice.NewScorer
	priorityExtractor domainservice.PriorityZeroExtractor
	assembler         domainservice.RecommendationAssembler
	defaults          model.RecommendationDefaults
}

func NewGenerateLearningUnitRecommendationsUseCase(
	txManager apprepo.TxManager,
	stateRepo apprepo.LearningStateSnapshotReadRepository,
	servingStateRepo apprepo.UserUnitServingStateRepository,
	runRepo apprepo.SchedulerRunRepository,
	backlogCalculator domainservice.BacklogCalculator,
	quotaAllocator domainservice.QuotaAllocator,
	reviewScorer domainservice.ReviewScorer,
	newScorer domainservice.NewScorer,
	priorityExtractor domainservice.PriorityZeroExtractor,
	assembler domainservice.RecommendationAssembler,
) GenerateLearningUnitRecommendationsUseCase {
	return GenerateLearningUnitRecommendationsUseCase{
		txManager:         txManager,
		stateRepo:         stateRepo,
		servingStateRepo:  servingStateRepo,
		runRepo:           runRepo,
		backlogCalculator: backlogCalculator,
		quotaAllocator:    quotaAllocator,
		reviewScorer:      reviewScorer,
		newScorer:         newScorer,
		priorityExtractor: priorityExtractor,
		assembler:         assembler,
		defaults:          model.DefaultRecommendationDefaults(),
	}
}

func (uc GenerateLearningUnitRecommendationsUseCase) Execute(ctx context.Context, cmd command.GenerateRecommendationsCommand) (dto.GenerateRecommendationsResult, error) {
	now := cmd.Now
	if now.IsZero() {
		now = time.Now()
	}

	requestedLimit := cmd.RequestedLimit
	if requestedLimit <= 0 {
		requestedLimit = uc.defaults.SessionDefaultLimit
	}

	reviewCandidates, err := uc.stateRepo.FindDueReviewCandidates(ctx, cmd.UserID, now)
	if err != nil {
		return dto.GenerateRecommendationsResult{}, err
	}
	newCandidates, err := uc.stateRepo.FindNewCandidates(ctx, cmd.UserID)
	if err != nil {
		return dto.GenerateRecommendationsResult{}, err
	}

	reviewBacklog := uc.backlogCalculator.Compute(len(reviewCandidates))
	quotas := uc.quotaAllocator.Allocate(reviewBacklog, requestedLimit, uc.defaults)

	scoredReviews := make([]appquery.ScoredReviewCandidate, 0, len(reviewCandidates))
	for _, candidate := range reviewCandidates {
		scoredReviews = append(scoredReviews, uc.reviewScorer.Score(candidate, now))
	}

	scoredNews := make([]appquery.ScoredNewCandidate, 0, len(newCandidates))
	for _, candidate := range newCandidates {
		scoredNews = append(scoredNews, uc.newScorer.Score(candidate, now))
	}

	priorityZero := uc.priorityExtractor.Extract(scoredReviews)
	batch := uc.assembler.Assemble(cmd.UserID, now, priorityZero, scoredReviews, scoredNews, quotas)
	batch.DueReviewCount = reviewBacklog

	err = uc.txManager.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.runRepo.SaveRun(ctx, batch); err != nil {
			return err
		}
		if err := uc.runRepo.SaveRunItems(ctx, batch); err != nil {
			return err
		}
		if err := uc.servingStateRepo.TouchRecommendedAt(ctx, batch.UserID, batch.RunID, coarseUnitIDs(batch), batch.GeneratedAt); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return dto.GenerateRecommendationsResult{}, err
	}

	return dto.GenerateRecommendationsResult{Batch: batch}, nil
}

func coarseUnitIDs(batch model.RecommendationBatch) []int64 {
	ids := make([]int64, 0, len(batch.Items))
	for _, item := range batch.Items {
		ids = append(ids, item.CoarseUnitID)
	}

	return ids
}
