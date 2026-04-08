// 作用：声明 learningengine 模块根包，作为整个学习状态引擎的包级入口说明。
// 输入/输出：不直接接收运行时输入，也不直接产生业务输出；主要提供包级语义。
// 谁调用它：IDE、godoc、阅读源码的维护者会首先看到这个文件。
// 它调用谁/传给谁：不调用其他文件；语义上为 application、domain、infrastructure、test 提供总入口。
package learningengine
