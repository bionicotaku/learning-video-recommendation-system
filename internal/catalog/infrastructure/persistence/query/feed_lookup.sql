-- name: ListFeedVideosByIDs :many
select
  v.video_id,
  v.title,
  coalesce(v.description, '')::text as description,
  v.video_object_path,
  v.thumbnail_url,
  coalesce(s.view_count, 0)::bigint as view_count,
  coalesce(s.like_count, 0)::bigint as like_count,
  coalesce(s.favorite_count, 0)::bigint as favorite_count
from catalog.videos v
left join catalog.video_engagement_stats s on s.video_id = v.video_id
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

