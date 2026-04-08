-- 文件作用：
--   - 创建 recommendation.scheduler_run_items 表，记录一轮 run 中每个 coarse unit 的推荐明细
-- 输入/输出：
--   - 输入：无，执行当前 migration up
--   - 输出：创建 run item 表
-- 谁调用它：
--   - migration 执行器
-- 它调用谁/传给谁：
--   - 直接传给 PostgreSQL 执行
create table if not exists recommendation.scheduler_run_items (
  run_id uuid not null references recommendation.scheduler_runs(run_id) on delete cascade,
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  recommend_type text not null check (recommend_type in ('review', 'new')),
  rank int not null,
  score numeric(8,4) not null,
  reason_codes text[] not null default '{}',
  primary key (run_id, coarse_unit_id)
);
