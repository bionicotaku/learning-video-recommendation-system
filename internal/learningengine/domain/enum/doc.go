// 作用：定义 Learning engine 使用的稳定枚举，避免业务代码散落硬编码字符串。
// 输入/输出：输入无；输出是事件类型、unit 类型、状态类型等枚举。
// 谁调用它：command、model、rule、service、aggregate、mapper、测试。
// 它调用谁/传给谁：不主动调用其他文件；枚举值会被整个模块传递和消费。
package enum
