insert into recommendation.user_unit_serving_states (
  user_id,
  coarse_unit_id,
  last_recommended_at,
  last_recommendation_run_id,
  created_at,
  updated_at
)
select
  s.user_id,
  s.coarse_unit_id,
  s.last_recommended_at,
  null,
  coalesce(s.updated_at, now()),
  coalesce(s.updated_at, now())
from learning.user_unit_states s
where s.last_recommended_at is not null
on conflict (user_id, coarse_unit_id) do update
set
  last_recommended_at = excluded.last_recommended_at,
  updated_at = excluded.updated_at;
