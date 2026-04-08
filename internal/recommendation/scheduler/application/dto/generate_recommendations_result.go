// 文件作用：
//   - 定义 GenerateRecommendationsResult，封装 scheduler 主用例的返回结果
//   - 当前只暴露 RecommendationBatch，保持返回面简洁
//
// 输入/输出：
//   - 输入：usecase 生成的 domain/model.RecommendationBatch
//   - 输出：返回给上层调用方的 DTO
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go 会构造并返回它
//   - 集成测试和场景测试会读取它的 Batch 字段
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为 usecase.Execute 的返回值传给上层调用方
package dto

import "learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

// GenerateRecommendationsResult returns the scheduler batch for downstream recommendation stages.
type GenerateRecommendationsResult struct {
	Batch model.RecommendationBatch
}
