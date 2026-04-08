// 作用：声明 Learning engine 的 use case 包，这一层是模块对外的应用入口。
// 输入/输出：输入来自 command；输出返回 dto。
// 谁调用它：上层业务调用方、集成测试、fixture。
// 它调用谁/传给谁：会调用 repository port、application service 和 domain aggregate。
package usecase
