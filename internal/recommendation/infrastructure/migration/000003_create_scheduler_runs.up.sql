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
