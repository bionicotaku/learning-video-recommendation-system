create table if not exists learning.user_unit_states (
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,

  is_target boolean not null default false,
  target_source text,
  target_source_ref_id text,
  target_priority numeric(8,4) not null default 0,

  status text not null default 'new'
    check (status in ('new', 'learning', 'reviewing', 'mastered', 'suspended')),

  progress_percent numeric(5,2) not null default 0
    check (progress_percent between 0 and 100),

  mastery_score numeric(5,4) not null default 0
    check (mastery_score between 0 and 1),

  first_observed_at timestamptz,
  last_observed_at timestamptz,
  observation_count integer not null default 0,

  progress_event_count integer not null default 0,
  last_progress_at timestamptz,
  last_progress_quality smallint check (last_progress_quality between 0 and 5),
  recent_progress_qualities smallint[] not null default '{}',
  recent_progress_passes boolean[] not null default '{}',
  progress_success_count integer not null default 0,
  progress_failure_count integer not null default 0,
  consecutive_success_count integer not null default 0,
  consecutive_failure_count integer not null default 0,

  schedule_repetition integer not null default 0,
  schedule_interval_days numeric(8,2) not null default 0,
  schedule_ease_factor numeric(6,4) not null default 2.5
    check (schedule_ease_factor >= 1.3),
  next_review_at timestamptz,

  suspended_reason text,

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  primary key (user_id, coarse_unit_id)
);
