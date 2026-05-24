create schema if not exists recommendation;

create table if not exists recommendation.user_unit_serving_states (
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  last_served_at timestamptz,
  last_run_id uuid,
  served_count integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  primary key (user_id, coarse_unit_id),
  check (served_count >= 0)
);

create table if not exists recommendation.user_video_serving_states (
  user_id uuid not null references auth.users(id) on delete cascade,
  video_id uuid not null references catalog.videos(video_id) on delete cascade,
  last_served_at timestamptz,
  last_run_id uuid,
  served_count integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  primary key (user_id, video_id),
  check (served_count >= 0)
);

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

create materialized view if not exists recommendation.v_video_unit_recall_index as
with scored as (
  select
    vui.video_id,
    vui.coarse_unit_id,
    vui.mention_count,
    vui.sentence_count,
    vui.coverage_ms,
    vui.coverage_ratio,
    vui.sentence_indexes,
    vui.best_evidence_sentence_index,
    vui.best_evidence_span_index,
    vui.best_evidence_start_ms,
    vui.best_evidence_end_ms,
    vui.best_evidence_scores,
    vui.best_evidence_question_reject_reason,
    vui.best_evidence_selection_reason,
    vui.best_evidence_candidate_score,
    vui.best_evidence_target_text,
    v.duration_ms,
    vt.mapped_span_ratio,
    v.status,
    v.visibility_status,
    v.publish_at,
    round((
      coalesce(vui.best_evidence_candidate_score, 0)::numeric / 10.0 * 0.45
      + vui.coverage_ratio * 0.25
      + least(vui.mention_count::numeric / 4.0, 1.0) * 0.15
      + least(vui.sentence_count::numeric / 3.0, 1.0) * 0.10
      + vt.mapped_span_ratio * 0.05
    ), 6)::numeric(10,6) as content_quality_score
  from catalog.video_unit_index as vui
  join catalog.videos as v on v.video_id = vui.video_id
  join catalog.video_transcripts as vt on vt.video_id = vui.video_id
  where v.status = 'active'
    and v.visibility_status = 'public'
    and (v.publish_at is null or v.publish_at <= now())
)
select
  scored.*,
  row_number() over (
    partition by coarse_unit_id
    order by content_quality_score desc, coverage_ratio desc, mention_count desc, video_id asc
  )::integer as rank_within_unit
from scored;

create materialized view if not exists recommendation.v_unit_video_inventory as
with recommendable as (
  select
    video_id,
    coarse_unit_id,
    mention_count,
    sentence_count,
    coverage_ms,
    coverage_ratio,
    mapped_span_ratio,
    content_quality_score
  from recommendation.v_video_unit_recall_index
),
aggregated as (
  select
    coarse_unit_id,
    count(distinct video_id)::integer as distinct_video_count,
    coalesce(avg(mention_count), 0)::numeric(10,4) as avg_mention_count,
    coalesce(avg(sentence_count), 0)::numeric(10,4) as avg_sentence_count,
    coalesce(avg(coverage_ms), 0)::numeric(12,4) as avg_coverage_ms,
    coalesce(avg(coverage_ratio), 0)::numeric(10,5) as avg_coverage_ratio,
    count(*) filter (
      where mention_count >= 2
        and coverage_ratio >= 0.05
        and mapped_span_ratio >= 0.50
        and content_quality_score >= 0.50
    )::integer as strong_video_count
  from recommendable
  group by coarse_unit_id
)
select
  a.coarse_unit_id,
  a.distinct_video_count,
  a.avg_mention_count,
  a.avg_sentence_count,
  a.avg_coverage_ms,
  a.avg_coverage_ratio,
  a.strong_video_count,
  case
    when a.strong_video_count >= 4 or a.distinct_video_count >= 8 then 'strong'
    when a.strong_video_count >= 2 or a.distinct_video_count >= 4 then 'ok'
    when a.distinct_video_count >= 1 then 'weak'
    else 'none'
  end as supply_grade,
  now()::timestamptz as updated_at
from aggregated as a;

create index if not exists idx_recommendation_unit_serving_states_last_served_at
on recommendation.user_unit_serving_states (user_id, last_served_at desc);

create index if not exists idx_recommendation_video_serving_states_last_served_at
on recommendation.user_video_serving_states (user_id, last_served_at desc);

create index if not exists idx_video_recommendation_runs_user_created_at
on recommendation.video_recommendation_runs (user_id, created_at desc);

create index if not exists idx_video_recommendation_items_video_id
on recommendation.video_recommendation_items (video_id);

create index if not exists idx_video_recommendation_items_dominant_unit
on recommendation.video_recommendation_items (dominant_unit_id)
where dominant_unit_id is not null;

create unique index if not exists idx_v_video_unit_recall_index_unit_video
on recommendation.v_video_unit_recall_index (coarse_unit_id, video_id);

create index if not exists idx_v_video_unit_recall_index_video_id
on recommendation.v_video_unit_recall_index (video_id);

create index if not exists idx_v_video_unit_recall_index_unit_rank
on recommendation.v_video_unit_recall_index (coarse_unit_id, rank_within_unit, video_id);

create index if not exists idx_v_video_unit_recall_index_unit_quality
on recommendation.v_video_unit_recall_index (
  coarse_unit_id,
  content_quality_score desc,
  coverage_ratio desc,
  mention_count desc,
  video_id
);

create unique index if not exists idx_v_unit_video_inventory_unit
on recommendation.v_unit_video_inventory (coarse_unit_id);

create index if not exists idx_v_unit_video_inventory_supply_grade
on recommendation.v_unit_video_inventory (supply_grade, coarse_unit_id);

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
