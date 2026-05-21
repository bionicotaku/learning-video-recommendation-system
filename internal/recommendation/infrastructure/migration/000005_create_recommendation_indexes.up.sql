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
  now() as updated_at
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
