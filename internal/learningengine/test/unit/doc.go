// 作用：声明 Learning engine 单元测试包，承载纯规则和轻量基础设施校验。
// 输入/输出：输入来自 go test 构造的伪状态、伪事件和伪配置；输出是单测结果。
// 谁调用它：go test、make check。
// 它调用谁/传给谁：调用 domain 层和基础设施层的纯函数或小对象。
package unit
