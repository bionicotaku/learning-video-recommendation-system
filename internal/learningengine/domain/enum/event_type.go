// 作用：定义标准化学习事件类型枚举，统一 strong/weak event 的合法取值。
// 输入/输出：输入无；输出是 EventType 类型和 exposure/lookup/new_learn/review/quiz 常量。
// 谁调用它：application/command、domain/rule、domain/aggregate、mapper、测试。
// 它调用谁/传给谁：不主动调用其他文件；事件类型会传给 reducer、SQL mapper 和测试断言。
package enum

type EventType string

const (
	EventTypeExposure EventType = "exposure"
	EventTypeLookup   EventType = "lookup"
	EventTypeNewLearn EventType = "new_learn"
	EventTypeReview   EventType = "review"
	EventTypeQuiz     EventType = "quiz"
)
