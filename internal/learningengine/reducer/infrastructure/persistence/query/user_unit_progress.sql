-- name: ListMasteredUnitProgress :many
select
  s.coarse_unit_id,
  cu.kind,
  cu.label,
  lower(cu.label)::text as label_key,
  cu.pos,
  cu.chinese_label,
  cu.chinese_def,
  s.progress_percent,
  s.last_progress_at
from learning.user_unit_states s
join semantic.coarse_unit cu
  on cu.id = s.coarse_unit_id
where s.user_id = sqlc.arg(user_id)
  and s.status = 'mastered'
  and cu.status = 'active'
  and (
    not sqlc.arg(has_cursor)::boolean
    or lower(cu.label) > sqlc.arg(cursor_label_key)::text
    or (
      lower(cu.label) = sqlc.arg(cursor_label_key)::text
      and cu.label > sqlc.arg(cursor_label)::text
    )
    or (
      lower(cu.label) = sqlc.arg(cursor_label_key)::text
      and cu.label = sqlc.arg(cursor_label)::text
      and s.coarse_unit_id > sqlc.arg(cursor_coarse_unit_id)::bigint
    )
  )
order by lower(cu.label) asc, cu.label asc, s.coarse_unit_id asc
limit sqlc.arg(limit_plus_one)::integer;

-- name: ListUnmasteredUnitProgress :many
select
  s.coarse_unit_id,
  cu.kind,
  cu.label,
  lower(cu.label)::text as label_key,
  cu.pos,
  cu.chinese_label,
  cu.chinese_def,
  s.progress_percent,
  s.last_progress_at
from learning.user_unit_states s
join semantic.coarse_unit cu
  on cu.id = s.coarse_unit_id
where s.user_id = sqlc.arg(user_id)
  and s.is_target = true
  and s.status in ('new', 'learning', 'reviewing')
  and cu.status = 'active'
  and (
    not sqlc.arg(has_cursor)::boolean
    or s.progress_percent < sqlc.arg(cursor_progress_percent)::numeric
    or (
      s.progress_percent = sqlc.arg(cursor_progress_percent)::numeric
      and lower(cu.label) > sqlc.arg(cursor_label_key)::text
    )
    or (
      s.progress_percent = sqlc.arg(cursor_progress_percent)::numeric
      and lower(cu.label) = sqlc.arg(cursor_label_key)::text
      and cu.label > sqlc.arg(cursor_label)::text
    )
    or (
      s.progress_percent = sqlc.arg(cursor_progress_percent)::numeric
      and lower(cu.label) = sqlc.arg(cursor_label_key)::text
      and cu.label = sqlc.arg(cursor_label)::text
      and s.coarse_unit_id > sqlc.arg(cursor_coarse_unit_id)::bigint
    )
  )
order by
  s.progress_percent desc,
  lower(cu.label) asc,
  cu.label asc,
  s.coarse_unit_id asc
limit sqlc.arg(limit_plus_one)::integer;
