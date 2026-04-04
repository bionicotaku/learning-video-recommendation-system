create table if not exists learning.scheduler_run_items (
  run_id uuid not null references learning.scheduler_runs(run_id) on delete cascade,
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,

  recommend_type text not null check (recommend_type in ('review', 'new')),
  rank int not null,
  score numeric(8,4) not null,
  reason_codes text[] not null default '{}',

  primary key (run_id, coarse_unit_id)
);
