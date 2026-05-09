create table if not exists recommendation.video_recommendation_runs (
  run_id uuid primary key,
  user_id uuid not null references auth.users(id) on delete cascade,
  request_context jsonb not null default '{}'::jsonb,
  session_mode text,
  selector_mode text,
  planner_snapshot jsonb not null default '{}'::jsonb,
  lane_budget_snapshot jsonb not null default '{}'::jsonb,
  candidate_summary jsonb not null default '{}'::jsonb,
  underfilled boolean not null default false,
  result_count integer not null default 0,
  created_at timestamptz not null default now(),

  check (result_count >= 0)
);

create table if not exists recommendation.video_recommendation_items (
  run_id uuid not null references recommendation.video_recommendation_runs(run_id) on delete cascade,
  rank integer not null,
  video_id uuid not null references catalog.videos(video_id) on delete cascade,
  score numeric(10,4) not null default 0,
  primary_lane text,
  dominant_role text,
  dominant_unit_id bigint,
  reason_codes text[] not null default '{}',
  learning_units jsonb not null default '[]'::jsonb,
  created_at timestamptz not null default now(),

  primary key (run_id, rank),
  foreign key (dominant_unit_id) references semantic.coarse_unit(id) on delete set null,
  check (rank > 0),
  check (jsonb_typeof(learning_units) = 'array')
);
