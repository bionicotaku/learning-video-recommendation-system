create table if not exists learning.scheduler_runs (
  run_id uuid primary key,
  user_id uuid not null references auth.users(id) on delete cascade,

  requested_limit int not null,
  generated_at timestamptz not null default now(),

  due_review_count int not null,
  selected_review_count int not null,
  selected_new_count int not null,

  context jsonb not null default '{}'::jsonb
);
