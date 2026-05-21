-- name: GetUserProfile :one
select user_id, email, email_confirmed_at, display_name, avatar_url, locale, timezone, onboarding_status, created_at, updated_at
from app_user.user_profiles
where user_id = $1;

-- name: GetAuthUser :one
select id, email, email_confirmed_at
from auth.users
where id = $1;

-- name: InsertRepairedUserProfile :one
insert into app_user.user_profiles (
  user_id,
  email,
  email_confirmed_at,
  display_name,
  locale,
  onboarding_status
) values (
  sqlc.arg(user_id),
  sqlc.arg(email),
  sqlc.arg(email_confirmed_at),
  nullif(split_part(coalesce(sqlc.arg(email)::text, ''), '@', 1), ''),
  'zh-CN',
  'new'
)
on conflict (user_id) do nothing
returning user_id, email, email_confirmed_at, display_name, avatar_url, locale, timezone, onboarding_status, created_at, updated_at;

-- name: UpdateUserTimezone :exec
update app_user.user_profiles
set timezone = sqlc.arg(timezone),
    updated_at = now()
where user_id = sqlc.arg(user_id)
  and timezone is distinct from sqlc.arg(timezone);

-- name: UpdateOnboardingStatus :exec
update app_user.user_profiles
set onboarding_status = sqlc.arg(onboarding_status),
    updated_at = now()
where user_id = sqlc.arg(user_id);

-- name: EnsureActivityStats :exec
insert into app_user.user_activity_stats (user_id)
values (sqlc.arg(user_id))
on conflict (user_id) do nothing;

-- name: GetActivityStats :one
select user_id, total_watch_ms, quiz_attempt_count, started_unit_count, updated_at
from app_user.user_activity_stats
where user_id = $1;

-- name: ListDailyActivityStats :many
select user_id, local_date, timezone, watch_ms, quiz_attempt_count, learning_interaction_count, first_activity_at, last_activity_at, updated_at
from app_user.user_daily_activity_stats
where user_id = sqlc.arg(user_id)
  and local_date between sqlc.arg(from_date)::date and sqlc.arg(to_date)::date
order by local_date asc;
