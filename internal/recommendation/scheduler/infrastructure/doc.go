// Package infrastructure 是 scheduler 的基础设施层。
//
// 文件作用：
//   - 承载配置、数据库连接、migration、SQL、mapper、repository 和事务实现
//   - 把 Recommendation 的业务规则与具体技术细节分离开
//
// 输入/输出：
//   - 输入来自 application/usecase 和 application/repository 接口约束
//   - 输出为数据库连接、仓储实现和事务实现
//
// 谁调用它：
//   - 外层组装代码和测试夹具会直接使用这里的构造函数
//   - application/usecase 通过接口间接依赖这里
//
// 它调用谁/传给谁：
//   - 调用 pgx、sqlc 生成层以及 PostgreSQL
package infrastructure
