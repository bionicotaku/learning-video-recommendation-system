-- name: AddWatchDuration :exec
with stats as (
  insert into app_user.user_activity_stats (
    user_id,
    total_watch_ms,
    updated_at
  ) values (
    sqlc.arg(user_id),
    greatest(sqlc.arg(delta_watch_ms)::bigint, 0),
    now()
  )
  on conflict (user_id) do update set
    total_watch_ms = app_user.user_activity_stats.total_watch_ms + greatest(sqlc.arg(delta_watch_ms)::bigint, 0),
    updated_at = now()
)
insert into app_user.user_daily_activity_stats (
  user_id,
  local_date,
  timezone,
  watch_ms,
  first_activity_at,
  last_activity_at,
  updated_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(local_date)::date,
  sqlc.arg(timezone),
  greatest(sqlc.arg(delta_watch_ms)::bigint, 0),
  sqlc.arg(activity_at),
  sqlc.arg(activity_at),
  now()
)
on conflict (user_id, local_date) do update set
  timezone = excluded.timezone,
  watch_ms = app_user.user_daily_activity_stats.watch_ms + excluded.watch_ms,
  first_activity_at = least(coalesce(app_user.user_daily_activity_stats.first_activity_at, excluded.first_activity_at), excluded.first_activity_at),
  last_activity_at = greatest(coalesce(app_user.user_daily_activity_stats.last_activity_at, excluded.last_activity_at), excluded.last_activity_at),
  updated_at = now();

-- name: IncrementQuizAttempt :exec
with stats as (
  insert into app_user.user_activity_stats (
    user_id,
    quiz_attempt_count,
    updated_at
  ) values (
    sqlc.arg(user_id),
    1,
    now()
  )
  on conflict (user_id) do update set
    quiz_attempt_count = app_user.user_activity_stats.quiz_attempt_count + 1,
    updated_at = now()
)
insert into app_user.user_daily_activity_stats (
  user_id,
  local_date,
  timezone,
  quiz_attempt_count,
  first_activity_at,
  last_activity_at,
  updated_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(local_date)::date,
  sqlc.arg(timezone),
  1,
  sqlc.arg(activity_at),
  sqlc.arg(activity_at),
  now()
)
on conflict (user_id, local_date) do update set
  timezone = excluded.timezone,
  quiz_attempt_count = app_user.user_daily_activity_stats.quiz_attempt_count + 1,
  first_activity_at = least(coalesce(app_user.user_daily_activity_stats.first_activity_at, excluded.first_activity_at), excluded.first_activity_at),
  last_activity_at = greatest(coalesce(app_user.user_daily_activity_stats.last_activity_at, excluded.last_activity_at), excluded.last_activity_at),
  updated_at = now();

-- name: IncrementStartedUnit :exec
insert into app_user.user_activity_stats (
  user_id,
  started_unit_count,
  updated_at
) values (
  sqlc.arg(user_id),
  1,
  now()
)
on conflict (user_id) do update set
  started_unit_count = app_user.user_activity_stats.started_unit_count + 1,
  updated_at = now();

-- name: IncrementLearningInteraction :exec
insert into app_user.user_daily_activity_stats (
  user_id,
  local_date,
  timezone,
  learning_interaction_count,
  first_activity_at,
  last_activity_at,
  updated_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(local_date)::date,
  sqlc.arg(timezone),
  1,
  sqlc.arg(activity_at),
  sqlc.arg(activity_at),
  now()
)
on conflict (user_id, local_date) do update set
  timezone = excluded.timezone,
  learning_interaction_count = app_user.user_daily_activity_stats.learning_interaction_count + 1,
  first_activity_at = least(coalesce(app_user.user_daily_activity_stats.first_activity_at, excluded.first_activity_at), excluded.first_activity_at),
  last_activity_at = greatest(coalesce(app_user.user_daily_activity_stats.last_activity_at, excluded.last_activity_at), excluded.last_activity_at),
  updated_at = now();
