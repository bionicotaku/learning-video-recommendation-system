-- 作用：创建用户学习状态投影表 learning.user_unit_states。
-- 输入/输出：输入无；输出是状态表及其约束。
-- 谁调用它：仓库级 migrate 流程。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；后续由 state repository、Recommendation 读链路消费。
create table if not exists learning.user_unit_states (
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,

  is_target boolean not null default true,
  target_source text,
  target_source_ref_id text,
  target_priority numeric(5,4) not null default 0.5,

  status text not null default 'new'
    check (status in ('new', 'learning', 'reviewing', 'mastered', 'suspended')),

  progress_percent numeric(5,2) not null default 0
    check (progress_percent between 0 and 100),

  mastery_score numeric(5,4) not null default 0
    check (mastery_score between 0 and 1),

  first_seen_at timestamptz,
  last_seen_at timestamptz,
  last_reviewed_at timestamptz,

  seen_count int not null default 0,
  strong_event_count int not null default 0,
  review_count int not null default 0,
  correct_count int not null default 0,
  wrong_count int not null default 0,
  consecutive_correct int not null default 0,
  consecutive_wrong int not null default 0,

  last_quality smallint check (last_quality between 0 and 5),
  recent_quality_window smallint[] not null default '{}',
  recent_correctness_window boolean[] not null default '{}',

  repetition int not null default 0,
  interval_days numeric(8,2) not null default 0,
  ease_factor numeric(6,4) not null default 2.5,
  next_review_at timestamptz,

  suspended_reason text,

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  primary key (user_id, coarse_unit_id)
);
