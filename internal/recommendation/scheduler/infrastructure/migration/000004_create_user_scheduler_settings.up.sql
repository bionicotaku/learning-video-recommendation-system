create table if not exists learning.user_scheduler_settings (
  user_id uuid primary key references auth.users(id) on delete cascade,

  session_default_limit int not null default 20,
  daily_new_unit_quota int not null default 8,
  daily_review_soft_limit int not null default 30,
  daily_review_hard_limit int not null default 60,

  timezone text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
