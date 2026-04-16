create materialized view if not exists recommendation.v_unit_video_inventory as
with recommendable as (
  select *
  from recommendation.v_recommendable_video_units
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
    )::integer as strong_video_count
  from recommendable
  group by coarse_unit_id
)
select
  cu.id as coarse_unit_id,
  coalesce(a.distinct_video_count, 0)::integer as distinct_video_count,
  coalesce(a.avg_mention_count, 0)::numeric(10,4) as avg_mention_count,
  coalesce(a.avg_sentence_count, 0)::numeric(10,4) as avg_sentence_count,
  coalesce(a.avg_coverage_ms, 0)::numeric(12,4) as avg_coverage_ms,
  coalesce(a.avg_coverage_ratio, 0)::numeric(10,5) as avg_coverage_ratio,
  coalesce(a.strong_video_count, 0)::integer as strong_video_count,
  case
    when coalesce(a.strong_video_count, 0) >= 4 or coalesce(a.distinct_video_count, 0) >= 8 then 'strong'
    when coalesce(a.strong_video_count, 0) >= 2 or coalesce(a.distinct_video_count, 0) >= 4 then 'ok'
    when coalesce(a.distinct_video_count, 0) >= 1 then 'weak'
    else 'none'
  end as supply_grade,
  now() as updated_at
from semantic.coarse_unit as cu
left join aggregated as a on a.coarse_unit_id = cu.id;

create index if not exists idx_recommendation_unit_serving_states_last_served_at
on recommendation.user_unit_serving_states (user_id, last_served_at desc);

create index if not exists idx_recommendation_video_serving_states_last_served_at
on recommendation.user_video_serving_states (user_id, last_served_at desc);

create index if not exists idx_video_recommendation_runs_user_created_at
on recommendation.video_recommendation_runs (user_id, created_at desc);

create index if not exists idx_video_recommendation_items_video_id
on recommendation.video_recommendation_items (video_id);

create unique index if not exists idx_v_recommendable_video_units_unit_video
on recommendation.v_recommendable_video_units (coarse_unit_id, video_id);

create index if not exists idx_v_recommendable_video_units_video_id
on recommendation.v_recommendable_video_units (video_id);

create unique index if not exists idx_v_unit_video_inventory_unit
on recommendation.v_unit_video_inventory (coarse_unit_id);

create index if not exists idx_v_unit_video_inventory_supply_grade
on recommendation.v_unit_video_inventory (supply_grade, coarse_unit_id);
