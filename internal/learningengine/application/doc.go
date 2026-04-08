// 作用：声明 application 层包，说明这一层只负责编排用例，不承载领域规则本身。
// 输入/输出：不直接处理业务输入输出；真正的输入输出定义落在 command 和 dto 子包。
// 谁调用它：IDE、godoc、阅读源码的维护者。
// 它调用谁/传给谁：语义上承接上层调用方输入，并把请求交给 usecase、repository port、application service。
package application
