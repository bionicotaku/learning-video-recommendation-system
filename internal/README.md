# `internal/` 代码总览

`internal/` 是当前项目的核心业务代码目录。

当前仓库在 `internal/` 下只保留两个平级模块：

1. `learningengine`
2. `recommendation`

这两个模块是当前 MVP 的主业务边界。它们不是父子关系，也不是一个模块内部的子包拆分，而是两个职责不同、owner 不同的内部模块。

## 1. 整体职责分工

### `internal/learningengine`

负责：

- 维护学习行为真相层 `learning.unit_learning_events`
- 维护学习状态投影层 `learning.user_unit_states`
- 接收标准化学习事件
- 按领域规则归约学习状态
- 提供 full replay

不负责：

- 生成推荐批次
- 维护推荐投放状态
- 维护推荐审计

### `internal/recommendation`

负责：

- 读取 Learning engine 的学习状态
- 读取 `semantic.coarse_unit`
- 生成当前推荐批次
- 维护 `recommendation.user_unit_serving_states`
- 维护 `recommendation.scheduler_runs`
- 维护 `recommendation.scheduler_run_items`

不负责：

- 写 `learning.unit_learning_events`
- 写 `learning.user_unit_states`
- replay 学习状态

## 2. 模块依赖关系

当前设计要求：

- `learningengine` 和 `recommendation` 完全平级
- `recommendation` 不 import `learningengine` 的内部实现
- 两边都不通过“直接调用对方 use case”来协作
- 它们通过数据库 owner 边界和各自的 repository / query 输入解耦

可以把关系理解成：

```text
Learning engine
  -> 写 learning.*

Recommendation
  -> 读 learning.*
  -> 写 recommendation.*
```

也就是说：

- Learning engine 产出学习域业务状态
- Recommendation 消费学习域业务状态

## 3. 当前目录结构

```text
internal/
  README.md
  learningengine/
    README.md
    application/
    domain/
    infrastructure/
    test/
  recommendation/
    README.md
    application/
    domain/
    infrastructure/
    test/
```

两个模块内部都采用同一种分层方式：

- `application`
- `domain`
- `infrastructure`
- `test`

这不是为了形式统一，而是为了明确：

- 规则在哪里
- 编排在哪里
- SQL / 数据库连接在哪里
- 集成测试在哪里

## 4. 推荐的阅读顺序

如果你是第一次接手这个仓库，建议按下面顺序看代码：

1. 先读 [internal/learningengine/README.md](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/README.md)
2. 再读 [internal/recommendation/README.md](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/README.md)
3. 看 `application/usecase`
4. 看 `domain/*`
5. 看 `infrastructure/persistence/query/*.sql`
6. 最后看 `test/integration/*`

这样阅读的好处是：

- 先知道边界
- 再知道入口
- 再看规则
- 最后再看 SQL 和测试验证

## 5. 修改代码时的判断标准

可以用下面这套判断标准快速决定改动应该落在哪个模块。

### 改动属于 Learning engine

如果你改的是这些内容，就应该落在 `internal/learningengine`：

- 学习事件类型
- 弱事件 / 强事件规则
- SM-2 更新
- 状态迁移
- `progress_percent`
- `mastery_score`
- replay
- `learning.unit_learning_events`
- `learning.user_unit_states`

### 改动属于 Recommendation

如果你改的是这些内容，就应该落在 `internal/recommendation`：

- candidate query
- backlog / quota
- review / new scorer
- 推荐批次组装
- `last_recommended_at`
- `recommendation.user_unit_serving_states`
- `recommendation.scheduler_runs`
- `recommendation.scheduler_run_items`

## 6. 不要做什么

以后继续维护时，不要再做这些事：

- 不要把 Learning engine 规则写回 Recommendation
- 不要让 Recommendation 重新拥有 `learning.*` 写权限
- 不要把 Recommendation 的投放字段塞回 `learning.user_unit_states`
- 不要跨模块直接 import 对方内部实现
- 不要重新引入第三套混合 owner 的 `scheduler` 模块

## 7. 对应文档

代码和文档的对应关系如下：

- 总览：[docs/README.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/README.md)
- 系统总览：[MVP 推荐系统整体设计文档.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/MVP%20推荐系统整体设计文档.md)
- Learning engine：[学习引擎设计文档.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习引擎设计文档.md)
- Recommendation 设计：[学习调度系统设计.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习调度系统设计.md)
- Recommendation 工程：[学习调度系统工程实现稿.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习调度系统工程实现稿.md)
- Recommendation 实施说明：[学习调度系统模块实施说明.md](/Users/evan/Downloads/learning-video-recommendation-system/docs/学习调度系统模块实施说明.md)
