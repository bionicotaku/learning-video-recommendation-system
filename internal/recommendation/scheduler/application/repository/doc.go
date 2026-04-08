// Package repository 定义 scheduler 应用层依赖的仓储接口。
//
// 文件作用：
//   - 把 usecase 需要的读取、写入和事务能力抽象成稳定接口
//   - 让 application/usecase 不依赖具体 pgx、sqlc 或 SQL 实现
//
// 输入/输出：
//   - 输入来自 usecase 的读取请求和写入请求
//   - 输出为候选数据、落库结果或事务执行结果
//
// 谁调用它：
//   - application/usecase 会依赖这些接口
//
// 它调用谁/传给谁：
//   - 接口本身不调用实现
//   - 具体实现位于 infrastructure/persistence/repository 和 infrastructure/persistence/tx
package repository
