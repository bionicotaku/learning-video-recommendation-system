// 文件作用：
//   - 定义 GenerateRecommendationsCommand，作为 scheduler 主用例的应用层输入对象
//   - 统一承接用户、请求条数和时间这三个核心入参
//
// 输入/输出：
//   - 输入：上层调用方传入的 UserID、RequestedLimit、Now
//   - 输出：提供给 GenerateLearningUnitRecommendationsUseCase.Execute 的命令对象
//
// 谁调用它：
//   - 外层业务组装代码
//   - 测试夹具 fixture.GenerateCmd
//   - 集成测试和场景测试
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为参数传给 application/usecase/generate_recommendations.go
package command

import (
	"time"

	"github.com/google/uuid"
)

// GenerateRecommendationsCommand requests one scheduler recommendation batch.
type GenerateRecommendationsCommand struct {
	UserID         uuid.UUID
	RequestedLimit int
	Now            time.Time
}
