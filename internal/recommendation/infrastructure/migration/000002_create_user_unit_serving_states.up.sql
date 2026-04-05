create table if not exists recommendation.user_unit_serving_states (
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  last_recommended_at timestamptz,
  last_recommendation_run_id uuid,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (user_id, coarse_unit_id)
);
