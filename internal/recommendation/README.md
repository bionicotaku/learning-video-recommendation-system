# Recommendation

`internal/recommendation` 是 Recommendation 模块根目录。

职责：

- 读取 Learning engine 的学习状态
- 生成推荐批次
- 维护 `recommendation.user_unit_serving_states`
- 维护 `recommendation.scheduler_runs`
- 维护 `recommendation.scheduler_run_items`

不负责：

- 写 `learning.unit_learning_events`
- 写 `learning.user_unit_states`
- replay 学习状态
