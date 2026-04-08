// 作用：定义用户-unit 学习状态枚举，统一 new/learning/reviewing/mastered/suspended 的合法值。
// 输入/输出：输入无；输出是 UnitStatus 类型和相关常量。
// 谁调用它：domain/model、domain/service/status_transitioner.go、domain/aggregate、mapper、测试。
// 它调用谁/传给谁：不主动调用其他文件；状态值会传给 reducer、repository 和测试断言。
package enum

type UnitStatus string

const (
	UnitStatusNew       UnitStatus = "new"
	UnitStatusLearning  UnitStatus = "learning"
	UnitStatusReviewing UnitStatus = "reviewing"
	UnitStatusMastered  UnitStatus = "mastered"
	UnitStatusSuspended UnitStatus = "suspended"
)
