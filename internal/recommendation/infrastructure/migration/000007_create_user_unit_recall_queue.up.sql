create table if not exists recommendation.recall_projection_metadata (
  projection_name text primary key,
  projection_updated_at timestamptz not null default now()
);

insert into recommendation.recall_projection_metadata (projection_name, projection_updated_at)
values ('video_unit_recall_index', now())
on conflict (projection_name) do nothing;

create table if not exists recommendation.user_unit_recall_queue (
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  status text not null check (status in ('new', 'learning', 'reviewing')),
  target_priority numeric(8,4) not null default 0,
  mastery_score numeric(5,4) not null default 0 check (mastery_score between 0 and 1),
  last_progress_quality smallint check (last_progress_quality between 0 and 5),
  next_review_at timestamptz,
  supply_grade text not null default 'none' check (supply_grade in ('none', 'weak', 'ok', 'strong')),
  state_updated_at timestamptz not null,
  source_version text not null,
  rebuilt_at timestamptz not null default now(),

  primary key (user_id, coarse_unit_id)
);

create table if not exists recommendation.user_unit_recall_queue_states (
  user_id uuid primary key references auth.users(id) on delete cascade,
  source_learning_max_updated_at timestamptz,
  source_projection_updated_at timestamptz not null,
  active_target_unit_count integer not null default 0 check (active_target_unit_count >= 0),
  rebuilt_at timestamptz not null default now()
);

create index if not exists idx_user_unit_recall_queue_user_status_priority
on recommendation.user_unit_recall_queue (
  user_id,
  status,
  target_priority desc,
  coarse_unit_id
);

create index if not exists idx_user_unit_recall_queue_user_next_review
on recommendation.user_unit_recall_queue (
  user_id,
  next_review_at,
  coarse_unit_id
);
