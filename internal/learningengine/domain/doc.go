// 作用：声明 Learning engine 的领域层包，聚合状态规则、模型、策略和计算器。
// 输入/输出：不直接处理外部输入输出；主要沉淀领域对象与规则。
// 谁调用它：application/usecase、application/service、测试。
// 它调用谁/传给谁：domain 内部互相组合；最终把规则能力暴露给 application 层。
package domain
