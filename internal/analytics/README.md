# Analytics

`analytics` 负责保存前端和产品交互产生的原始事实。

当前模块边界包括：

- 习题 / 练习答题原始事实
- 观看 session 的低频摘要事实
- exposure、lookup、已学会等非答题学习互动原始事实
- raw fact 表统一的客户端环境上下文 `client_context`

当前已落地结构：

```text
internal/analytics/
  README.md
  doc.go
  application/
    dto/
    repository/
    service/
    usecase/
  domain/
    model/
  infrastructure/
    migration/
    persistence/
      mapper/
      query/
      repository/
      schema/
      sqlcgen/
      sqlc.yaml
  test/
```

当前已实现 raw fact write 能力：

- `RecordLearningInteractionsBatch` 应用用例负责 learning interaction 整批 validation。
- `RecordQuizAttempt` 应用用例负责单次 completed quiz attempt validation。
- quiz raw fact 写入 `analytics.quiz_events`。
- exposure / lookup / self_mark raw fact 写入 `analytics.learning_interaction_events`。
- `(user_id, client_event_id)` 重复时返回已有 `event_id`，不把重复当错误。
- 真实 repository 分别写入 quiz 与 learning interaction raw facts；两类事件由未来不同 API 调用，不再混入同一事务。
- `user_id` 来自 usecase request；未来 HTTP 层必须从认证上下文传入，不能信任事件 payload。
- `client_context` 只要求是 JSON object，不固定字段集合；当前 API 样例推荐四个基础字段，但后端不拒绝扩展字段。
- `shown_at`、`completed_at`、`occurred_at` 在 application/service 边界会归一化为 UTC instant；persistence mapper 通过 `internal/platform/postgres/pgtime` 统一写入 `timestamptz`。
- UUID、nullable text 等纯 Postgres 类型转换委托 `internal/platform/postgres/*`；Analytics 仍保留本地 mapper 函数作为模块边界。
- Integration fixture 使用 `internal/platform/postgres/pgtest` 管理 embedded Postgres 和 template database；Analytics 自己的 `test/fixture` 只声明 Analytics schema plan 与 seed helper。

Analytics 不负责：

- 计算 `progress_quality`
- 生成 `reducer_effect`
- 写入 `learning.*`
- 调用 reducer

raw fact 到 Learning Engine event 的解释代码放在 `internal/learningengine/normalizer`。
