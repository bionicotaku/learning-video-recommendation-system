-- 文件作用：
--   - 创建 Recommendation owner 的 user_unit_serving_states 表
-- 输入/输出：
--   - 输入：无，执行当前 migration up
--   - 输出：创建用户-学习单元级的最近推荐状态表
-- 谁调用它：
--   - migration 执行器
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
create table if not exists recommendation.user_unit_serving_states (
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  last_recommended_at timestamptz,
  last_recommendation_run_id uuid,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (user_id, coarse_unit_id)
);
