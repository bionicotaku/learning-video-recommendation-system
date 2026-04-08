// 作用：定义 application 层依赖的 repository 和 transaction port interface。
// 输入/输出：输入是 use case 希望执行的读写和事务需求；输出是抽象接口，不包含具体实现。
// 谁调用它：application/usecase。
// 它调用谁/传给谁：不主动调用其他文件；接口由 infrastructure/persistence/repository 和 infrastructure/persistence/tx 实现。
package repository
