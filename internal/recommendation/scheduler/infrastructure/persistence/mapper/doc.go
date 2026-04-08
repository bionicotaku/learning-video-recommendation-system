// Package mapper 定义 scheduler 持久化层的映射逻辑。
//
// 文件作用：
//   - 把 sqlc 生成类型和领域/应用层结构解耦
//   - 收口 pgtype 转换、枚举解析和落库参数构造
//
// 输入/输出：
//   - 输入来自 sqlc 查询结果或领域层 batch/状态对象
//   - 输出为 application/query、domain/model 或 sqlc 参数结构
//
// 谁调用它：
//   - infrastructure/persistence/repository/*.go
//
// 它调用谁/传给谁：
//   - 调用 pgtype helper 和 enum 解析函数
//   - 把转换结果传给 repository 或 sqlcgen
package mapper
