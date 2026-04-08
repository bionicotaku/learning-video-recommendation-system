// 作用：放置可复用的领域计算器，如 SM-2、状态迁移、进度和掌握分计算。
// 输入/输出：输入通常是 state、quality、recent window、policy；输出是计算后的数值或被修改的 state。
// 谁调用它：domain/aggregate/user_unit_reducer.go、测试。
// 它调用谁/传给谁：处理结果会传回 reducer；部分 service 内部调用本文件自身的辅助函数。
package service
