// 作用：声明 Learning engine 的基础设施层，负责数据库、事务、SQL、mapper 和 migration 的技术落地。
// 输入/输出：输入来自 application 层的持久化需求；输出是 repository、tx manager、DB pool 等具体实现。
// 谁调用它：启动装配代码、application/usecase、集成测试、fixture。
// 它调用谁/传给谁：调用 pgx、sqlc 生成代码和 PostgreSQL；实现结果传给 application 层使用。
package infrastructure
