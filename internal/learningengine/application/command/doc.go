// 作用：定义 application 层的命令对象，用稳定结构承接外部请求参数。
// 输入/输出：输入是上层调用方的原始业务参数；输出是供 use case 消费的 command struct。
// 谁调用它：上层调用方、集成测试、fixture。
// 它调用谁/传给谁：不主动调用其他文件；生成的命令对象会传给 application/usecase。
package command
