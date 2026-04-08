// 作用：放置原子规则处理器和状态初始化辅助函数，是 reducer 的第一层规则积木。
// 输入/输出：输入通常是 current state 和 event；输出是更新后的 state 或 error。
// 谁调用它：domain/aggregate/user_unit_reducer.go、测试。
// 它调用谁/传给谁：会把处理后的 state 传回 reducer；部分文件内部调用 state_helpers.go。
package rule
