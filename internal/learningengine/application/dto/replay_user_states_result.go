// 作用：定义 replay 用例的返回结果，报告成功重建了多少状态以及是否发生错误。
// 输入/输出：输入是 replay 执行过程中得到的 rebuilt count 和 error count；输出是 ReplayUserStatesResult 结构。
// 谁调用它：application/usecase/replay_user_states.go。
// 它调用谁/传给谁：不主动调用其他文件；会作为返回值传给上层调用方和集成测试。
package dto

type ReplayUserStatesResult struct {
	RebuiltCount int
	ErrorCount   int
}
