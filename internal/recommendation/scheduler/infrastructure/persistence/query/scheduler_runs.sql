-- 文件作用：
--   - 定义 scheduler_runs 和 scheduler_run_items 的查询与写入 SQL
-- 输入/输出：
--   - 输入：RecommendationBatch 映射出的 run / item 参数
--   - 输出：写入 recommendation.scheduler_runs 与 recommendation.scheduler_run_items，或返回计数结果
-- 谁调用它：
--   - sqlc 读取它生成 CountSchedulerRuns / UpsertSchedulerRun / UpsertSchedulerRunItem
--   - repository/scheduler_run_repo.go 和测试间接调用
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
-- name: CountSchedulerRuns :one
select count(*)::bigint
from recommendation.scheduler_runs;

-- name: UpsertSchedulerRun :exec
insert into recommendation.scheduler_runs (
  run_id,
  user_id,
  requested_limit,
  generated_at,
  due_review_count,
  selected_review_count,
  selected_new_count,
  context
) values (
  sqlc.arg(run_id),
  sqlc.arg(user_id),
  sqlc.arg(requested_limit),
  sqlc.arg(generated_at),
  sqlc.arg(due_review_count),
  sqlc.arg(selected_review_count),
  sqlc.arg(selected_new_count),
  sqlc.arg(context)
)
on conflict (run_id) do update
set
  user_id = excluded.user_id,
  requested_limit = excluded.requested_limit,
  generated_at = excluded.generated_at,
  due_review_count = excluded.due_review_count,
  selected_review_count = excluded.selected_review_count,
  selected_new_count = excluded.selected_new_count,
  context = excluded.context;

-- name: UpsertSchedulerRunItem :exec
insert into recommendation.scheduler_run_items (
  run_id,
  user_id,
  coarse_unit_id,
  recommend_type,
  rank,
  score,
  reason_codes
) values (
  sqlc.arg(run_id),
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.arg(recommend_type),
  sqlc.arg(rank),
  sqlc.arg(score),
  sqlc.arg(reason_codes)
)
on conflict (run_id, coarse_unit_id) do update
set
  user_id = excluded.user_id,
  recommend_type = excluded.recommend_type,
  rank = excluded.rank,
  score = excluded.score,
  reason_codes = excluded.reason_codes;
