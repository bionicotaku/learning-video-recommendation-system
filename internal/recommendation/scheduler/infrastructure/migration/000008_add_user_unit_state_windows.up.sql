alter table learning.user_unit_states
  add column if not exists recent_quality_window smallint[] not null default '{}',
  add column if not exists recent_correctness_window boolean[] not null default '{}';
