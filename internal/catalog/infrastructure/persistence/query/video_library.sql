-- name: ListVideoFavorites :many
select
  v.video_id,
  v.title,
  v.thumbnail_url,
  v.duration_ms,
  coalesce(s.view_count, 0)::bigint as view_count,
  state.bookmarked_at
from catalog.video_user_states state
join catalog.videos v on v.video_id = state.video_id
left join catalog.video_engagement_stats s on s.video_id = state.video_id
where state.user_id = sqlc.arg(user_id)::uuid
  and state.has_bookmarked = true
  and state.bookmarked_at is not null
  and v.status = 'active'
  and v.visibility_status = 'public'
  and (v.publish_at is null or v.publish_at <= now())
  and (
    not sqlc.arg(has_cursor)::boolean
    or state.bookmarked_at < sqlc.arg(cursor_at)::timestamptz
    or (
      state.bookmarked_at = sqlc.arg(cursor_at)::timestamptz
      and state.video_id > sqlc.arg(cursor_video_id)::uuid
    )
  )
order by state.bookmarked_at desc, state.video_id asc
limit sqlc.arg(limit_plus_one)::int;

-- name: ListVideoHistory :many
select
  v.video_id,
  v.title,
  v.thumbnail_url,
  v.duration_ms,
  coalesce(s.view_count, 0)::bigint as view_count,
  coalesce(state.last_position_ms, 0)::integer as last_position_ms,
  state.last_watched_at
from catalog.video_user_states state
join catalog.videos v on v.video_id = state.video_id
left join catalog.video_engagement_stats s on s.video_id = state.video_id
where state.user_id = sqlc.arg(user_id)::uuid
  and state.has_watched = true
  and state.last_watched_at is not null
  and v.status = 'active'
  and v.visibility_status = 'public'
  and (v.publish_at is null or v.publish_at <= now())
  and (
    not sqlc.arg(has_cursor)::boolean
    or state.last_watched_at < sqlc.arg(cursor_at)::timestamptz
    or (
      state.last_watched_at = sqlc.arg(cursor_at)::timestamptz
      and state.video_id > sqlc.arg(cursor_video_id)::uuid
    )
  )
order by state.last_watched_at desc, state.video_id asc
limit sqlc.arg(limit_plus_one)::int;
