-- name: ListFeedVideosByIDs :many
select
  v.video_id,
  v.title,
  coalesce(v.description, '')::text as description,
  v.video_object_path,
  v.thumbnail_url,
  transcript.transcript_object_path,
  coalesce(s.view_count, 0)::bigint as view_count,
  coalesce(s.like_count, 0)::bigint as like_count,
  coalesce(s.favorite_count, 0)::bigint as favorite_count,
  coalesce(user_state.has_liked, false)::boolean as has_liked,
  coalesce(user_state.has_bookmarked, false)::boolean as has_favorited
from catalog.videos v
left join catalog.video_engagement_stats s on s.video_id = v.video_id
left join catalog.video_transcripts transcript on transcript.video_id = v.video_id
left join catalog.video_user_states user_state
  on user_state.user_id = sqlc.arg(user_id)::uuid
 and user_state.video_id = v.video_id
where v.video_id = any(sqlc.arg(video_ids)::uuid[])
  and v.status = 'active'
  and v.visibility_status = 'public'
  and (v.publish_at is null or v.publish_at <= now())
order by v.video_id;

-- name: ListUnitLabelsByIDs :many
select
  id,
  label
from semantic.coarse_unit
where id = any(sqlc.arg(coarse_unit_ids)::bigint[])
  and status = 'active'
order by id;
