-- 作用：创建学习事件真相层表 learning.unit_learning_events。
-- 输入/输出：输入无；输出是事件表及其约束。
-- 谁调用它：仓库级 migrate 流程。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；后续由 event repository 和 replay 链路消费。
create table if not exists learning.unit_learning_events (
  event_id bigserial primary key,

  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  video_id uuid references catalog.videos(video_id) on delete set null,

  event_type text not null
    check (event_type in ('exposure', 'lookup', 'new_learn', 'review', 'quiz')),

  source_type text not null,
  source_ref_id text,

  is_correct boolean,
  quality smallint check (quality between 0 and 5),
  response_time_ms int,
  metadata jsonb not null default '{}'::jsonb,

  occurred_at timestamptz not null,
  created_at timestamptz not null default now()
);
