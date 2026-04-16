# Learning Engine

`learningengine` 负责学习事件、学习状态、target/control 字段和 replay 相关边界。

当前首轮落地只补：

- 模块标准骨架
- Learning engine owner 的 migration
- SQL/query/sqlc 基础层
- DTO、repository port 和 usecase 接口

当前不补：

- reducer / replay 业务逻辑
- SM-2 / mastery / progress 规则实现
- 任何 Recommendation 相关投放逻辑
