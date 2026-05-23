-- name: ListPendingLearningInteractions :many
select
  i.event_id,
  i.user_id,
  i.event_type,
  i.source_surface,
  i.video_id,
  i.watch_session_id,
  i.recommendation_run_id,
  i.related_quiz_event_id,
  i.coarse_unit_id,
  i.token_text,
  i.sentence_index,
  i.span_index,
  i.occurred_at,
  i.exposure_start_ms,
  i.exposure_end_ms,
  i.exposure_count,
  i.lookup_visible_ms,
  i.lookup_sentence_audio_replay_count,
  i.lookup_word_audio_play_count,
  i.lookup_practice_now_clicked,
  i.event_payload
from analytics.learning_interaction_events i
where i.coarse_unit_id is not null
  and i.event_type in ('lookup', 'self_mark_mastered')
  and (sqlc.narg(user_id)::uuid is null or i.user_id = sqlc.narg(user_id)::uuid)
  and (sqlc.narg(occurred_before)::timestamptz is null or i.occurred_at < sqlc.narg(occurred_before)::timestamptz)
  and not exists (
    select 1
    from learning.unit_learning_events e
    where e.user_id = i.user_id
      and e.source_type = 'learning_interaction_event'
      and e.source_ref_id = i.event_id::text
      and e.coarse_unit_id = i.coarse_unit_id
  )
order by i.occurred_at asc, i.event_id asc
limit sqlc.arg(limit_count)::int;

-- name: ListLearningInteractionsByIDs :many
select
  i.event_id,
  i.user_id,
  i.event_type,
  i.source_surface,
  i.video_id,
  i.watch_session_id,
  i.recommendation_run_id,
  i.related_quiz_event_id,
  i.coarse_unit_id,
  i.token_text,
  i.sentence_index,
  i.span_index,
  i.occurred_at,
  i.exposure_start_ms,
  i.exposure_end_ms,
  i.exposure_count,
  i.lookup_visible_ms,
  i.lookup_sentence_audio_replay_count,
  i.lookup_word_audio_play_count,
  i.lookup_practice_now_clicked,
  i.event_payload
from analytics.learning_interaction_events i
where i.user_id = sqlc.arg(user_id)
  and i.event_type in ('exposure', 'lookup', 'self_mark_mastered')
  and i.event_id = any(sqlc.arg(event_ids)::uuid[])
order by i.occurred_at asc, i.event_id asc;

-- name: ListPendingExposureSession3Windows :many
with candidate_pairs as (
  select distinct i.user_id, i.coarse_unit_id
  from analytics.learning_interaction_events i
  where i.event_type = 'exposure'
    and i.coarse_unit_id is not null
    and i.watch_session_id is not null
    and (sqlc.narg(user_id)::uuid is null or i.user_id = sqlc.narg(user_id)::uuid)
    and (sqlc.narg(occurred_before)::timestamptz is null or i.occurred_at < sqlc.narg(occurred_before)::timestamptz)
),
latest_lookup as (
  select
    p.user_id,
    p.coarse_unit_id,
    coalesce(max(l.occurred_at), '-infinity'::timestamptz) as latest_lookup_at
  from candidate_pairs p
  left join analytics.learning_interaction_events l
    on l.user_id = p.user_id
   and l.coarse_unit_id = p.coarse_unit_id
   and l.event_type = 'lookup'
  group by p.user_id, p.coarse_unit_id
),
consumed_sessions as (
  select
    p.user_id,
    p.coarse_unit_id,
    consumed.watch_session_id
  from candidate_pairs p
  join learning.unit_learning_events e
    on e.user_id = p.user_id
   and e.coarse_unit_id = p.coarse_unit_id
   and e.source_type = 'exposure_session3_v1'
  cross join lateral unnest(e.consumed_watch_session_ids) as consumed(watch_session_id)
),
session_exposures as (
  select
    l.user_id,
    l.coarse_unit_id,
    i.watch_session_id,
    (array_agg(i.video_id order by i.occurred_at asc, i.event_id asc))[1] as video_id,
    min(i.occurred_at) as first_exposed_at,
    count(*)::integer as raw_event_count
  from latest_lookup l
  join analytics.learning_interaction_events i
    on i.user_id = l.user_id
   and i.coarse_unit_id = l.coarse_unit_id
   and i.event_type = 'exposure'
   and i.watch_session_id is not null
   and i.occurred_at > l.latest_lookup_at
   and (sqlc.narg(occurred_before)::timestamptz is null or i.occurred_at < sqlc.narg(occurred_before)::timestamptz)
  left join consumed_sessions c
    on c.user_id = l.user_id
   and c.coarse_unit_id = l.coarse_unit_id
   and c.watch_session_id = i.watch_session_id
  where c.watch_session_id is null
  group by l.user_id, l.coarse_unit_id, i.watch_session_id
),
ranked as (
  select
    s.*,
    row_number() over (
      partition by s.user_id, s.coarse_unit_id
      order by s.first_exposed_at asc, s.watch_session_id asc
    ) as session_rank
  from session_exposures s
),
windowed as (
  select
    r.*,
    ((r.session_rank - 1) / 3)::integer as window_index
  from ranked r
),
windows as (
  select
    w.user_id,
    w.coarse_unit_id,
    (array_agg(w.video_id::text order by w.session_rank asc))[3] as third_video_id,
    max(w.first_exposed_at) as occurred_at,
    array_agg(w.watch_session_id::text order by w.session_rank asc) as watch_session_ids,
    array_agg(w.video_id::text order by w.session_rank asc) as video_ids,
    sum(w.raw_event_count)::integer as raw_event_count
  from windowed w
  group by w.user_id, w.coarse_unit_id, w.window_index
  having count(*) = 3
)
select
  w.user_id,
  w.coarse_unit_id,
  w.occurred_at::timestamptz as occurred_at,
  w.third_video_id::text as third_video_id,
  w.watch_session_ids::text[] as watch_session_ids,
  w.video_ids::text[] as video_ids,
  w.raw_event_count
from windows w
order by w.occurred_at asc, w.user_id asc, w.coarse_unit_id asc
limit sqlc.arg(limit_count)::int;

-- name: ListExposureSession3WindowsByIDs :many
with candidate_pairs as (
  select distinct i.user_id, i.coarse_unit_id
  from analytics.learning_interaction_events i
  where i.user_id = sqlc.arg(user_id)
    and i.event_type = 'exposure'
    and i.coarse_unit_id is not null
    and i.watch_session_id is not null
    and i.event_id = any(sqlc.arg(event_ids)::uuid[])
),
latest_lookup as (
  select
    p.user_id,
    p.coarse_unit_id,
    coalesce(max(l.occurred_at), '-infinity'::timestamptz) as latest_lookup_at
  from candidate_pairs p
  left join analytics.learning_interaction_events l
    on l.user_id = p.user_id
   and l.coarse_unit_id = p.coarse_unit_id
   and l.event_type = 'lookup'
  group by p.user_id, p.coarse_unit_id
),
consumed_sessions as (
  select
    p.user_id,
    p.coarse_unit_id,
    consumed.watch_session_id
  from candidate_pairs p
  join learning.unit_learning_events e
    on e.user_id = p.user_id
   and e.coarse_unit_id = p.coarse_unit_id
   and e.source_type = 'exposure_session3_v1'
  cross join lateral unnest(e.consumed_watch_session_ids) as consumed(watch_session_id)
),
session_exposures as (
  select
    l.user_id,
    l.coarse_unit_id,
    i.watch_session_id,
    (array_agg(i.video_id order by i.occurred_at asc, i.event_id asc))[1] as video_id,
    min(i.occurred_at) as first_exposed_at,
    count(*)::integer as raw_event_count
  from latest_lookup l
  join analytics.learning_interaction_events i
    on i.user_id = l.user_id
   and i.coarse_unit_id = l.coarse_unit_id
   and i.event_type = 'exposure'
   and i.watch_session_id is not null
   and i.occurred_at > l.latest_lookup_at
  left join consumed_sessions c
    on c.user_id = l.user_id
   and c.coarse_unit_id = l.coarse_unit_id
   and c.watch_session_id = i.watch_session_id
  where c.watch_session_id is null
  group by l.user_id, l.coarse_unit_id, i.watch_session_id
),
ranked as (
  select
    s.*,
    row_number() over (
      partition by s.user_id, s.coarse_unit_id
      order by s.first_exposed_at asc, s.watch_session_id asc
    ) as session_rank
  from session_exposures s
),
windowed as (
  select
    r.*,
    ((r.session_rank - 1) / 3)::integer as window_index
  from ranked r
),
windows as (
  select
    w.user_id,
    w.coarse_unit_id,
    (array_agg(w.video_id::text order by w.session_rank asc))[3] as third_video_id,
    max(w.first_exposed_at) as occurred_at,
    array_agg(w.watch_session_id::text order by w.session_rank asc) as watch_session_ids,
    array_agg(w.video_id::text order by w.session_rank asc) as video_ids,
    sum(w.raw_event_count)::integer as raw_event_count
  from windowed w
  group by w.user_id, w.coarse_unit_id, w.window_index
  having count(*) = 3
)
select
  w.user_id,
  w.coarse_unit_id,
  w.occurred_at::timestamptz as occurred_at,
  w.third_video_id::text as third_video_id,
  w.watch_session_ids::text[] as watch_session_ids,
  w.video_ids::text[] as video_ids,
  w.raw_event_count
from windows w
order by w.occurred_at asc, w.user_id asc, w.coarse_unit_id asc;
