# Recommendation

`internal/recommendation` 是 Recommendation 模块根目录。

职责：

- 读取 Learning engine 的学习状态
- 生成推荐批次
- 维护 `recommendation.user_unit_serving_states`
- 维护 `recommendation.scheduler_runs`
- 维护 `recommendation.scheduler_run_items`
- 使用模块内默认调度参数生成推荐

不负责：

- 写 `learning.unit_learning_events`
- 写 `learning.user_unit_states`
- replay 学习状态
- 维护用户级调度配置表

MVP 明确约束：

- Recommendation 不支持用户级调度配置
- `session_default_limit`、`daily_new_unit_quota`、`daily_review_soft_limit`、`daily_review_hard_limit`、`timezone` 当前统一使用模块默认值

后续如果需要扩展，应由 Recommendation 自己新增配置表，不回写 Learning engine。
