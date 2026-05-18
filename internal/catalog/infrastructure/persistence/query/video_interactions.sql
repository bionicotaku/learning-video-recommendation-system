-- name: SetVideoLiked :one
with target_video as (
  select video_id
  from catalog.videos
  where video_id = sqlc.arg(video_id)::uuid
    and status = 'active'
    and visibility_status = 'public'
    and (publish_at is null or publish_at <= now())
),
changed_state as (
  insert into catalog.video_user_states (
    user_id,
    video_id,
    has_liked,
    liked_at,
    updated_at
  )
  select
    sqlc.arg(user_id)::uuid,
    target_video.video_id,
    true,
    now(),
    now()
  from target_video
  on conflict (user_id, video_id) do update set
    has_liked = true,
    liked_at = now(),
    updated_at = now()
  where catalog.video_user_states.has_liked = false
  returning video_id
),
delta as (
  select count(*)::bigint as value
  from changed_state
),
upsert_stats as (
  insert into catalog.video_engagement_stats (
    video_id,
    like_count,
    updated_at
  )
  select
    target_video.video_id,
    delta.value,
    now()
  from target_video
  cross join delta
  on conflict (video_id) do update set
    like_count = catalog.video_engagement_stats.like_count + excluded.like_count,
    updated_at = case
      when excluded.like_count > 0 then now()
      else catalog.video_engagement_stats.updated_at
    end
  returning video_id, like_count
)
select
  target_video.video_id,
  true::boolean as has_liked,
  coalesce(upsert_stats.like_count, stats.like_count, 0)::bigint as like_count
from target_video
left join upsert_stats on upsert_stats.video_id = target_video.video_id
left join catalog.video_engagement_stats stats on stats.video_id = target_video.video_id;

-- name: SetVideoUnliked :one
with target_video as (
  select video_id
  from catalog.videos
  where video_id = sqlc.arg(video_id)::uuid
    and status = 'active'
    and visibility_status = 'public'
    and (publish_at is null or publish_at <= now())
),
changed_state as (
  update catalog.video_user_states state
  set
    has_liked = false,
    liked_at = null,
    updated_at = now()
  from target_video
  where state.user_id = sqlc.arg(user_id)::uuid
    and state.video_id = target_video.video_id
    and state.has_liked = true
  returning state.video_id
),
delta as (
  select count(*)::bigint as value
  from changed_state
),
upsert_stats as (
  insert into catalog.video_engagement_stats (
    video_id,
    like_count,
    updated_at
  )
  select
    target_video.video_id,
    0,
    now()
  from target_video
  on conflict (video_id) do update set
    like_count = greatest(0, catalog.video_engagement_stats.like_count - (select value from delta)),
    updated_at = case
      when (select value from delta) > 0 then now()
      else catalog.video_engagement_stats.updated_at
    end
  returning video_id, like_count
)
select
  target_video.video_id,
  false::boolean as has_liked,
  coalesce(upsert_stats.like_count, stats.like_count, 0)::bigint as like_count
from target_video
left join upsert_stats on upsert_stats.video_id = target_video.video_id
left join catalog.video_engagement_stats stats on stats.video_id = target_video.video_id;

-- name: SetVideoFavorited :one
with target_video as (
  select video_id
  from catalog.videos
  where video_id = sqlc.arg(video_id)::uuid
    and status = 'active'
    and visibility_status = 'public'
    and (publish_at is null or publish_at <= now())
),
changed_state as (
  insert into catalog.video_user_states (
    user_id,
    video_id,
    has_bookmarked,
    bookmarked_at,
    updated_at
  )
  select
    sqlc.arg(user_id)::uuid,
    target_video.video_id,
    true,
    now(),
    now()
  from target_video
  on conflict (user_id, video_id) do update set
    has_bookmarked = true,
    bookmarked_at = now(),
    updated_at = now()
  where catalog.video_user_states.has_bookmarked = false
  returning video_id
),
delta as (
  select count(*)::bigint as value
  from changed_state
),
upsert_stats as (
  insert into catalog.video_engagement_stats (
    video_id,
    favorite_count,
    updated_at
  )
  select
    target_video.video_id,
    delta.value,
    now()
  from target_video
  cross join delta
  on conflict (video_id) do update set
    favorite_count = catalog.video_engagement_stats.favorite_count + excluded.favorite_count,
    updated_at = case
      when excluded.favorite_count > 0 then now()
      else catalog.video_engagement_stats.updated_at
    end
  returning video_id, favorite_count
)
select
  target_video.video_id,
  true::boolean as has_favorited,
  coalesce(upsert_stats.favorite_count, stats.favorite_count, 0)::bigint as favorite_count
from target_video
left join upsert_stats on upsert_stats.video_id = target_video.video_id
left join catalog.video_engagement_stats stats on stats.video_id = target_video.video_id;

-- name: SetVideoUnfavorited :one
with target_video as (
  select video_id
  from catalog.videos
  where video_id = sqlc.arg(video_id)::uuid
    and status = 'active'
    and visibility_status = 'public'
    and (publish_at is null or publish_at <= now())
),
changed_state as (
  update catalog.video_user_states state
  set
    has_bookmarked = false,
    bookmarked_at = null,
    updated_at = now()
  from target_video
  where state.user_id = sqlc.arg(user_id)::uuid
    and state.video_id = target_video.video_id
    and state.has_bookmarked = true
  returning state.video_id
),
delta as (
  select count(*)::bigint as value
  from changed_state
),
upsert_stats as (
  insert into catalog.video_engagement_stats (
    video_id,
    favorite_count,
    updated_at
  )
  select
    target_video.video_id,
    0,
    now()
  from target_video
  on conflict (video_id) do update set
    favorite_count = greatest(0, catalog.video_engagement_stats.favorite_count - (select value from delta)),
    updated_at = case
      when (select value from delta) > 0 then now()
      else catalog.video_engagement_stats.updated_at
    end
  returning video_id, favorite_count
)
select
  target_video.video_id,
  false::boolean as has_favorited,
  coalesce(upsert_stats.favorite_count, stats.favorite_count, 0)::bigint as favorite_count
from target_video
left join upsert_stats on upsert_stats.video_id = target_video.video_id
left join catalog.video_engagement_stats stats on stats.video_id = target_video.video_id;
