// 作用：定义持久化映射层，隔离 domain model 与 sqlc/pgtype 生成类型。
// 输入/输出：输入是 domain 对象或 sqlc row/params；输出是相反方向的转换结果。
// 谁调用它：persistence/repository 下的两个仓储实现。
// 它调用谁/传给谁：调用 pgtype_helpers.go；转换结果传给 sqlc querier 或 domain 层。
package mapper
