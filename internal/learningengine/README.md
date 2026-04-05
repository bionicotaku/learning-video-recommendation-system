# Learning Engine

`internal/learningengine` 是 Learning engine 模块根目录。

职责：

- 维护 `learning.unit_learning_events`
- 维护 `learning.user_unit_states`
- 处理标准化学习事件
- 提供 full replay

不负责：

- 生成推荐批次
- 维护 `last_recommended_at`
- 维护推荐运行审计
