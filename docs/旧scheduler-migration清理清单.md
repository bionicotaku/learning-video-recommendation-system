# 旧 scheduler 清理说明

本文档记录这次重构中，旧 `scheduler` 结构已经如何被清理，以及对于**已有数据库**还需要执行哪些一次性 SQL。

## 1. 已完成的仓库级清理

仓库内已完成：

- 删除 `internal/recommendation/scheduler`
- 删除旧 scheduler migration 根
- 删除旧 scheduler sqlc 配置块
- 删除旧 scheduler 测试与 README
- 删除旧 `Makefile` 中的 `scheduler-migrate-*` 命令

当前仓库只保留两个 owner：

- [internal/learningengine](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine)
- [internal/recommendation](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation)

## 2. 新的数据库 owner

当前最终 owner 如下：

| 对象 | owner |
| --- | --- |
| `learning.unit_learning_events` | Learning engine |
| `learning.user_unit_states` | Learning engine |
| `recommendation.user_unit_serving_states` | Recommendation |
| `recommendation.scheduler_runs` | Recommendation |
| `recommendation.scheduler_run_items` | Recommendation |

已经不再保留的旧 owner：

- `learning.user_unit_states.last_recommended_at`
- `learning.scheduler_runs`
- `learning.scheduler_run_items`
- `learning.user_scheduler_settings`

## 3. 现有数据库的一次性清理脚本

如果某个数据库曾经跑过旧 scheduler migration，需要额外执行两份一次性 SQL。

### 3.1 先回填 serving state

脚本：

- [backfill_recommendation_serving_states.sql](/Users/evan/Downloads/learning-video-recommendation-system/scripts/legacy/backfill_recommendation_serving_states.sql)

作用：

- 把旧 `learning.user_unit_states.last_recommended_at` 中已有数据迁到
  `recommendation.user_unit_serving_states`

### 3.2 再删除旧对象

脚本：

- [drop_old_scheduler_objects.sql](/Users/evan/Downloads/learning-video-recommendation-system/scripts/legacy/drop_old_scheduler_objects.sql)

作用：

- 删除 `learning.user_unit_states.last_recommended_at`
- 删除 `learning.scheduler_runs`
- 删除 `learning.scheduler_run_items`
- 删除 `learning.user_scheduler_settings`

## 4. 建议执行顺序

对于已有数据库，推荐顺序是：

1. 运行 Learning engine 新 migration
2. 运行 Recommendation 新 migration
3. 执行 `backfill_recommendation_serving_states.sql`
4. 执行 `drop_old_scheduler_objects.sql`

## 5. 当前结论

从代码仓库角度看，旧 scheduler 结构已经彻底删除。

从数据库角度看：

- 新库：直接使用 Learning engine 和 Recommendation 两套新 migration
- 旧库：额外执行一次性清理脚本后，与新结构一致
