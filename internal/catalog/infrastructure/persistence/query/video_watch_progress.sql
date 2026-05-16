-- name: GetVideoDurationMS :one
select duration_ms
from catalog.videos
where video_id = sqlc.arg(video_id);

-- name: UpsertVideoWatchSession :one
with input as (
  select
    sqlc.arg(watch_session_id)::uuid as watch_session_id,
    sqlc.arg(user_id)::uuid as user_id,
    sqlc.arg(video_id)::uuid as video_id,
    sqlc.arg(occurred_at)::timestamptz as occurred_at,
    sqlc.arg(position_ms)::integer as position_ms,
    sqlc.arg(active_watch_ms)::bigint as active_watch_ms,
    sqlc.arg(duration_ms)::integer as duration_ms,
    sqlc.arg(client_context)::jsonb as client_context,
    sqlc.arg(metadata)::jsonb as metadata
),
existing as (
  select e.*
  from analytics.video_watch_events e
  join input i on i.watch_session_id = e.watch_session_id
  for update
),
computed_update as (
  select
    i.watch_session_id,
    i.user_id,
    i.video_id,
    i.occurred_at,
    i.position_ms,
    i.active_watch_ms,
    i.duration_ms,
    i.client_context,
    i.metadata,
    e.active_watch_ms as old_active_watch_ms,
    e.is_completed as old_is_completed,
    greatest(e.max_position_ms, i.position_ms) as new_max_position_ms,
    greatest(e.active_watch_ms, i.active_watch_ms) as new_active_watch_ms,
    (
      greatest(e.active_watch_ms, i.active_watch_ms) > 10000
      and i.duration_ms > 0
      and (greatest(e.max_position_ms, i.position_ms)::numeric / i.duration_ms::numeric) >= 0.9
    ) as computed_completed
  from input i
  join existing e
    on e.watch_session_id = i.watch_session_id
   and e.user_id = i.user_id
   and e.video_id = i.video_id
),
inserted as (
  insert into analytics.video_watch_events (
    watch_session_id,
    user_id,
    video_id,
    started_at,
    last_seen_at,
    completed_at,
    last_position_ms,
    max_position_ms,
    active_watch_ms,
    is_completed,
    progress_report_count,
    client_context,
    metadata
  )
  select
    i.watch_session_id,
    i.user_id,
    i.video_id,
    i.occurred_at,
    i.occurred_at,
    case
      when i.active_watch_ms > 10000
       and i.duration_ms > 0
       and (i.position_ms::numeric / i.duration_ms::numeric) >= 0.9
      then i.occurred_at
      else null
    end,
    i.position_ms,
    i.position_ms,
    i.active_watch_ms,
    (
      i.active_watch_ms > 10000
      and i.duration_ms > 0
      and (i.position_ms::numeric / i.duration_ms::numeric) >= 0.9
    ),
    1,
    i.client_context,
    i.metadata
  from input i
  where not exists (select 1 from existing)
  on conflict (watch_session_id) do nothing
  returning
    true::boolean as created_session,
    is_completed as completed_session,
    active_watch_ms as delta_active_watch_ms,
    started_at,
    last_seen_at,
    last_position_ms,
    max_position_ms,
    active_watch_ms,
    is_completed
),
updated as (
  update analytics.video_watch_events e
  set
    last_seen_at = greatest(e.last_seen_at, c.occurred_at),
    last_position_ms = case
      when c.occurred_at >= e.last_seen_at then c.position_ms
      else e.last_position_ms
    end,
    max_position_ms = c.new_max_position_ms,
    active_watch_ms = c.new_active_watch_ms,
    is_completed = e.is_completed or c.computed_completed,
    completed_at = coalesce(
      e.completed_at,
      case
        when not e.is_completed and c.computed_completed then c.occurred_at
        else null
      end
    ),
    progress_report_count = e.progress_report_count + 1,
    client_context = c.client_context,
    metadata = c.metadata,
    updated_at = now()
  from computed_update c
  where e.watch_session_id = c.watch_session_id
  returning
    false::boolean as created_session,
    (not c.old_is_completed and c.computed_completed)::boolean as completed_session,
    (c.new_active_watch_ms - c.old_active_watch_ms)::bigint as delta_active_watch_ms,
    e.started_at,
    e.last_seen_at,
    e.last_position_ms,
    e.max_position_ms,
    e.active_watch_ms,
    e.is_completed
)
select *
from inserted
union all
select *
from updated
limit 1;

-- name: UpsertVideoUserStateFromWatchProgress :exec
insert into catalog.video_user_states (
  user_id,
  video_id,
  has_watched,
  first_watched_at,
  last_watched_at,
  watch_count,
  completed_count,
  last_position_ms,
  max_position_ms,
  total_watch_ms
) values (
  sqlc.arg(user_id),
  sqlc.arg(video_id),
  true,
  sqlc.arg(started_at),
  sqlc.arg(last_seen_at),
  case when sqlc.arg(created_session)::boolean then 1 else 0 end,
  case when sqlc.arg(completed_session)::boolean then 1 else 0 end,
  sqlc.arg(last_position_ms),
  sqlc.arg(max_position_ms),
  sqlc.arg(delta_active_watch_ms)
)
on conflict (user_id, video_id) do update set
  has_watched = true,
  first_watched_at = coalesce(catalog.video_user_states.first_watched_at, excluded.first_watched_at),
  last_watched_at = greatest(
    coalesce(catalog.video_user_states.last_watched_at, excluded.last_watched_at),
    excluded.last_watched_at
  ),
  watch_count = catalog.video_user_states.watch_count + case when sqlc.arg(created_session)::boolean then 1 else 0 end,
  completed_count = catalog.video_user_states.completed_count + case when sqlc.arg(completed_session)::boolean then 1 else 0 end,
  last_position_ms = case
    when catalog.video_user_states.last_watched_at is null
      or excluded.last_watched_at >= catalog.video_user_states.last_watched_at
    then excluded.last_position_ms
    else catalog.video_user_states.last_position_ms
  end,
  max_position_ms = greatest(catalog.video_user_states.max_position_ms, excluded.max_position_ms),
  total_watch_ms = catalog.video_user_states.total_watch_ms + sqlc.arg(delta_active_watch_ms),
  updated_at = now();

-- name: UpsertVideoEngagementStatsFromWatchProgress :exec
insert into catalog.video_engagement_stats (
  video_id,
  view_count,
  completed_count,
  total_watch_ms
) values (
  sqlc.arg(video_id),
  case when sqlc.arg(created_session)::boolean then 1 else 0 end,
  case when sqlc.arg(completed_session)::boolean then 1 else 0 end,
  sqlc.arg(delta_active_watch_ms)
)
on conflict (video_id) do update set
  view_count = catalog.video_engagement_stats.view_count + case when sqlc.arg(created_session)::boolean then 1 else 0 end,
  completed_count = catalog.video_engagement_stats.completed_count + case when sqlc.arg(completed_session)::boolean then 1 else 0 end,
  total_watch_ms = catalog.video_engagement_stats.total_watch_ms + sqlc.arg(delta_active_watch_ms),
  updated_at = now();
