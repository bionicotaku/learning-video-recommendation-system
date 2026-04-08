-- 文件作用：
--   - 创建 recommendation.scheduler_runs 表，记录一轮 scheduler 执行的批次级审计
-- 输入/输出：
--   - 输入：无，执行当前 migration up
--   - 输出：创建 run 头表
-- 谁调用它：
--   - migration 执行器
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
create table if not exists recommendation.scheduler_runs (
  run_id uuid primary key,
  user_id uuid not null references auth.users(id) on delete cascade,
  requested_limit int not null,
  generated_at timestamptz not null,
  due_review_count int not null default 0,
  selected_review_count int not null default 0,
  selected_new_count int not null default 0,
  context jsonb not null default '{}'::jsonb
);
