// 作用：定义记录学习事件用例的返回结果，报告接受了多少事件、影响了哪些 unit。
// 输入/输出：输入是 use case 在事务中累计得到的 accepted count 和 updated unit list；输出是 RecordLearningEventsResult 结构。
// 谁调用它：application/usecase/record_learning_events.go。
// 它调用谁/传给谁：不主动调用其他文件；会作为返回值传给上层调用方和集成测试。
package dto

type RecordLearningEventsResult struct {
	AcceptedCount int
	UpdatedUnits  []int64
}
