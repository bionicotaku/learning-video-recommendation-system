// 作用：定义 application 层的输出 DTO，用稳定结构向调用方返回 use case 执行结果。
// 输入/输出：输入来自 use case 内部计算结果；输出是给上层调用方使用的结果结构。
// 谁调用它：application/usecase、集成测试、阅读源码的维护者。
// 它调用谁/传给谁：不主动调用其他文件；DTO 会被 use case 返回给上层。
package dto
