// 作用：实现 application 层定义的 repository port，把 domain 对象真正落到 PostgreSQL。
// 输入/输出：输入是 use case 传入的查询或写入请求；输出是 domain 对象、error 或持久化副作用。
// 谁调用它：application/usecase、fixture/helpers.go。
// 它调用谁/传给谁：调用 mapper、queryctx、sqlcgen 生成的 Querier；结果再传回 application 层。
package repository
