// Package model 定义 scheduler 领域层的核心模型。
//
// 文件作用：
//   - 承载跨层传递的领域对象和结果对象
//   - 让 usecase、domain/service、repository mapper 在同一套模型上协作
//
// 输入/输出：
//   - 输入来自 application/query、mapper 和 assembler
//   - 输出为最终 RecommendationBatch 或中间领域对象
//
// 谁调用它：
//   - application/usecase
//   - domain/service
//   - infrastructure/persistence/mapper
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
package model
