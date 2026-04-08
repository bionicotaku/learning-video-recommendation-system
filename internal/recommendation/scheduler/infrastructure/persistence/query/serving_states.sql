-- 文件作用：
--   - 定义 Recommendation 自有 serving state 的 upsert SQL
-- 输入/输出：
--   - 输入：user_id、coarse_unit_id、last_recommended_at、last_recommendation_run_id 等参数
--   - 输出：插入或更新 recommendation.user_unit_serving_states
-- 谁调用它：
--   - sqlc 读取它生成 UpsertUserUnitServingState
--   - repository/user_unit_serving_state_repo.go 间接调用
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
-- name: UpsertUserUnitServingState :exec
insert into recommendation.user_unit_serving_states (
  user_id,
  coarse_unit_id,
  last_recommended_at,
  last_recommendation_run_id,
  created_at,
  updated_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.arg(last_recommended_at),
  sqlc.arg(last_recommendation_run_id),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
on conflict (user_id, coarse_unit_id) do update
set
  last_recommended_at = excluded.last_recommended_at,
  last_recommendation_run_id = excluded.last_recommendation_run_id,
  updated_at = excluded.updated_at;
