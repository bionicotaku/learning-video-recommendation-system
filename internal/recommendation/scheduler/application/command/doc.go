// Package command 定义 scheduler 应用层命令对象。
//
// 文件作用：
//   - 作为 usecase 的显式输入边界
//   - 把调用方传入的参数整理成稳定结构，避免直接传散乱参数
//
// 输入/输出：
//   - 输入是上层调用方希望执行一次推荐所需的原始参数
//   - 输出是交给 application/usecase 的 command 结构体
//
// 谁调用它：
//   - 上层调用方、测试夹具和集成测试会构造这里的命令对象
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 主要传给 usecase.Execute
package command
