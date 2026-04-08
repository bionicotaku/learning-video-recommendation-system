// 作用：集中定义 Learning engine 的固定策略参数，避免魔法数字散落在规则实现中。
// 输入/输出：输入无；输出是 LearningPolicy 及默认值。
// 谁调用它：application/usecase、application/service、domain/aggregate、domain/service、测试。
// 它调用谁/传给谁：不主动调用其他文件；策略对象会传给 reducer 和各类计算器。
package policy
