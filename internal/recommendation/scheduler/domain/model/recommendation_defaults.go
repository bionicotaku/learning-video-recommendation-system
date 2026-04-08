// 文件作用：
//   - 定义 RecommendationDefaults 和默认配置构造函数
//   - 承接当前 MVP 不支持用户级配置时的固定调度参数
//
// 输入/输出：
//   - 输入：无，默认值由代码内固定给出
//   - 输出：供 usecase 和 quota allocator 使用的默认配置结构
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go 在构造 usecase 时加载默认值
//   - 测试也会直接构造或复用该结构
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为参数传给 QuotaAllocator
package model

// RecommendationDefaults contains the fixed MVP recommendation limits.
type RecommendationDefaults struct {
	SessionDefaultLimit  int
	DailyNewUnitQuota    int
	DailyReviewSoftLimit int
	DailyReviewHardLimit int
	Timezone             string
}

// DefaultRecommendationDefaults returns the fixed MVP recommendation defaults.
func DefaultRecommendationDefaults() RecommendationDefaults {
	return RecommendationDefaults{
		SessionDefaultLimit:  20,
		DailyNewUnitQuota:    8,
		DailyReviewSoftLimit: 30,
		DailyReviewHardLimit: 60,
	}
}
