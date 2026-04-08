// 作用：定义“按用户 full replay 重建状态”用例的输入命令。
// 输入/输出：输入只有 UserID；输出是 ReplayUserStatesCommand 结构定义。
// 谁调用它：上层业务调用方、集成测试、fixture/helpers.go。
// 它调用谁/传给谁：不主动调用其他文件；命令对象会传给 application/usecase/replay_user_states.go。
package command

import "github.com/google/uuid"

// ReplayUserStatesCommand requests a full user-state rebuild from event history.
type ReplayUserStatesCommand struct {
	UserID uuid.UUID
}
