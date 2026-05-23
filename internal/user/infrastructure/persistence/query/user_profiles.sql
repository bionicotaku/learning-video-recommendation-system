-- name: GetUserProfile :one
select user_id, email, email_confirmed_at, display_name, avatar_url, locale, timezone, onboarding_status, birth_date, gender, education_stage, ip_region, created_at, updated_at
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
  coalesce(nullif(split_part(coalesce(sqlc.arg(email)::text, ''), '@', 1), ''), 'user'),
  'zh-CN',
  'new'
)
on conflict (user_id) do nothing
returning user_id, email, email_confirmed_at, display_name, avatar_url, locale, timezone, onboarding_status, birth_date, gender, education_stage, ip_region, created_at, updated_at;

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

-- name: GetCurrentActivityStreakDays :one
with recursive anchor as (
  select candidate.local_date
  from (
    select sqlc.arg(today)::date as local_date
    union all
    select (sqlc.arg(today)::date - 1)::date as local_date
  ) candidate
  where exists (
    select 1
    from app_user.user_daily_activity_stats s
    where s.user_id = sqlc.arg(user_id)
      and s.local_date = candidate.local_date
      and (
        s.watch_ms > 0
        or s.quiz_attempt_count > 0
        or s.learning_interaction_count > 0
      )
  )
  order by candidate.local_date desc
  limit 1
),
streak(local_date) as (
  select local_date
  from anchor
  union all
  select (streak.local_date - 1)::date
  from streak
  where exists (
    select 1
    from app_user.user_daily_activity_stats s
    where s.user_id = sqlc.arg(user_id)
      and s.local_date = (streak.local_date - 1)::date
      and (
        s.watch_ms > 0
        or s.quiz_attempt_count > 0
        or s.learning_interaction_count > 0
      )
  )
)
select count(*)::bigint
from streak;
