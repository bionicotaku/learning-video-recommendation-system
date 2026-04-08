// 作用：定义 Learning engine 的核心领域模型，让业务规则围绕事件和状态对象展开。
// 输入/输出：输入来自 use case 和 mapper；输出供 rule、service、aggregate、repository 使用。
// 谁调用它：application/usecase、domain/rule、domain/service、mapper、测试。
// 它调用谁/传给谁：模型本身不主动调用其他文件；会在模块内部持续传递。
package model
