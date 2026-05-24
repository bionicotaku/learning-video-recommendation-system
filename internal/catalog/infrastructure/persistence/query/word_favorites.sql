-- name: HasWordFavoriteByCoarse :one
select exists (
  select 1
  from catalog.word_favorites
  where user_id = sqlc.arg(user_id)::uuid
    and favorite_key_type = 'coarse_unit'
    and coarse_unit_id = sqlc.arg(coarse_unit_id)::bigint
    and is_favorited = true
);

-- name: HasWordFavoriteByToken :one
select exists (
  select 1
  from catalog.word_favorites
  where user_id = sqlc.arg(user_id)::uuid
    and favorite_key_type = 'video_token'
    and video_id = sqlc.arg(video_id)::uuid
    and sentence_index = sqlc.arg(sentence_index)::integer
    and token_index = sqlc.arg(token_index)::integer
    and is_favorited = true
);

-- name: GetWordFavoriteVideoContext :one
select
  v.title,
  v.duration_ms,
  sentence.text as sentence_text,
  sentence.translation as sentence_translation,
  sentence.start_ms as sentence_start_ms,
  sentence.end_ms as sentence_end_ms
from catalog.videos v
join catalog.video_transcript_sentences sentence
  on sentence.video_id = v.video_id
where v.video_id = sqlc.arg(video_id)::uuid
  and sentence.sentence_index = sqlc.arg(sentence_index)::integer
  and v.status = 'active'
  and v.visibility_status = 'public'
  and (v.publish_at is null or v.publish_at <= now());

-- name: SetCoarseWordFavorite :one
with input as (
  select
    sqlc.arg(user_id)::uuid as user_id,
    sqlc.arg(coarse_unit_id)::bigint as coarse_unit_id,
    sqlc.arg(source)::text as source,
    sqlc.narg(video_id)::uuid as video_id,
    sqlc.narg(sentence_index)::integer as sentence_index,
    sqlc.narg(token_index)::integer as token_index,
    sqlc.arg(occurred_at)::timestamptz as occurred_at
),
existing as (
  select wf.state_updated_at, wf.is_favorited
  from catalog.word_favorites wf
  join input i
    on i.user_id = wf.user_id
   and i.coarse_unit_id = wf.coarse_unit_id
  where wf.favorite_key_type = 'coarse_unit'
  for update
),
state as (
  select
    exists (
      select 1
      from existing
      where state_updated_at > (select occurred_at from input)
    ) as is_stale,
    exists (
      select 1
      from existing
      where state_updated_at = (select occurred_at from input)
        and is_favorited = true
    ) as is_duplicate_set,
    exists (
      select 1
      from semantic.coarse_unit cu
      join input i on i.coarse_unit_id = cu.id
      where cu.status = 'active'
    ) as target_exists
),
upsert as (
insert into catalog.word_favorites (
  user_id,
  favorite_key_type,
  coarse_unit_id,
  source,
  video_id,
  sentence_index,
  token_index,
  is_favorited,
  favorited_at,
  state_updated_at
)
select
  i.user_id,
  'coarse_unit',
  i.coarse_unit_id,
  i.source,
  i.video_id,
  i.sentence_index,
  i.token_index,
  true,
  i.occurred_at,
  i.occurred_at
from input i
cross join state s
where not s.is_stale
  and not s.is_duplicate_set
  and s.target_exists
on conflict (user_id, coarse_unit_id) where favorite_key_type = 'coarse_unit'
do update set
  source = excluded.source,
  video_id = excluded.video_id,
  sentence_index = excluded.sentence_index,
  token_index = excluded.token_index,
  is_favorited = true,
  favorited_at = case
    when catalog.word_favorites.is_favorited then catalog.word_favorites.favorited_at
    else excluded.favorited_at
  end,
  state_updated_at = excluded.state_updated_at,
  updated_at = now()
where catalog.word_favorites.state_updated_at <= excluded.state_updated_at
returning 1
)
select case
  when (select is_stale from state) then 'stale'
  when (select is_duplicate_set from state) then 'applied'
  when not (select target_exists from state) then 'target_not_found'
  when exists (select 1 from upsert) then 'applied'
  else 'stale'
end::text as outcome;

-- name: SetTokenWordFavorite :one
with input as (
  select
    sqlc.arg(user_id)::uuid as user_id,
    sqlc.arg(video_id)::uuid as video_id,
    sqlc.arg(sentence_index)::integer as sentence_index,
    sqlc.arg(token_index)::integer as token_index,
    sqlc.arg(occurred_at)::timestamptz as occurred_at
),
existing as (
  select wf.state_updated_at, wf.is_favorited
  from catalog.word_favorites wf
  join input i
    on i.user_id = wf.user_id
   and i.video_id = wf.video_id
   and i.sentence_index = wf.sentence_index
   and i.token_index = wf.token_index
  where wf.favorite_key_type = 'video_token'
  for update
),
state as (
  select
    exists (
      select 1
      from existing
      where state_updated_at > (select occurred_at from input)
    ) as is_stale,
    exists (
      select 1
      from existing
      where state_updated_at = (select occurred_at from input)
        and is_favorited = true
    ) as is_duplicate_set,
    exists (
      select 1
      from catalog.videos v
      join input i on i.video_id = v.video_id
      join catalog.video_transcript_sentences sentence
        on sentence.video_id = v.video_id
       and sentence.sentence_index = i.sentence_index
      join catalog.video_semantic_spans span
        on span.video_id = sentence.video_id
       and span.sentence_index = sentence.sentence_index
       and span.span_index = i.token_index
      where v.status = 'active'
        and v.visibility_status = 'public'
        and (v.publish_at is null or v.publish_at <= now())
    ) as target_exists
),
upsert as (
insert into catalog.word_favorites (
  user_id,
  favorite_key_type,
  coarse_unit_id,
  source,
  video_id,
  sentence_index,
  token_index,
  is_favorited,
  favorited_at,
  state_updated_at
)
select
  i.user_id,
  'video_token',
  null,
  'video_transcript',
  i.video_id,
  i.sentence_index,
  i.token_index,
  true,
  i.occurred_at,
  i.occurred_at
from input i
cross join state s
where not s.is_stale
  and not s.is_duplicate_set
  and s.target_exists
on conflict (user_id, video_id, sentence_index, token_index) where favorite_key_type = 'video_token'
do update set
  is_favorited = true,
  favorited_at = case
    when catalog.word_favorites.is_favorited then catalog.word_favorites.favorited_at
    else excluded.favorited_at
  end,
  state_updated_at = excluded.state_updated_at,
  updated_at = now()
where catalog.word_favorites.state_updated_at <= excluded.state_updated_at
returning 1
)
select case
  when (select is_stale from state) then 'stale'
  when (select is_duplicate_set from state) then 'applied'
  when not (select target_exists from state) then 'target_not_found'
  when exists (select 1 from upsert) then 'applied'
  else 'stale'
end::text as outcome;

-- name: UnsetWordFavoriteByCoarse :exec
insert into catalog.word_favorites (
  user_id,
  favorite_key_type,
  coarse_unit_id,
  source,
  video_id,
  sentence_index,
  token_index,
  is_favorited,
  favorited_at,
  state_updated_at
)
values (
  sqlc.arg(user_id)::uuid,
  'coarse_unit',
  sqlc.arg(coarse_unit_id)::bigint,
  sqlc.arg(source)::text,
  sqlc.narg(video_id)::uuid,
  sqlc.narg(sentence_index)::integer,
  sqlc.narg(token_index)::integer,
  false,
  null::timestamptz,
  sqlc.arg(occurred_at)::timestamptz
)
on conflict (user_id, coarse_unit_id) where favorite_key_type = 'coarse_unit'
do update set
  source = excluded.source,
  video_id = excluded.video_id,
  sentence_index = excluded.sentence_index,
  token_index = excluded.token_index,
  is_favorited = false,
  favorited_at = null,
  state_updated_at = excluded.state_updated_at,
  updated_at = now()
where catalog.word_favorites.state_updated_at <= excluded.state_updated_at;

-- name: UnsetWordFavoriteByToken :exec
insert into catalog.word_favorites (
  user_id,
  favorite_key_type,
  coarse_unit_id,
  source,
  video_id,
  sentence_index,
  token_index,
  is_favorited,
  favorited_at,
  state_updated_at
) values (
  sqlc.arg(user_id)::uuid,
  'video_token',
  null,
  'video_transcript',
  sqlc.arg(video_id)::uuid,
  sqlc.arg(sentence_index)::integer,
  sqlc.arg(token_index)::integer,
  false,
  null,
  sqlc.arg(occurred_at)::timestamptz
)
on conflict (user_id, video_id, sentence_index, token_index) where favorite_key_type = 'video_token'
do update set
  is_favorited = false,
  favorited_at = null,
  state_updated_at = excluded.state_updated_at,
  updated_at = now()
where catalog.word_favorites.state_updated_at <= excluded.state_updated_at;

-- name: ListWordFavorites :many
select
  wf.favorite_id,
  wf.favorited_at,
  wf.coarse_unit_id,
  cu.label,
  cu.pos,
  cu.chinese_label,
  cu.chinese_def,
  wf.source,
  wf.video_id,
  wf.sentence_index,
  wf.token_index,
  span.surface_text as source_text,
  sentence.translation as source_translation,
  span.dictionary as source_dictionary,
  span.explanation as source_explanation
from catalog.word_favorites wf
left join semantic.coarse_unit cu
  on cu.id = wf.coarse_unit_id
 and cu.status = 'active'
left join catalog.videos source_video
  on source_video.video_id = wf.video_id
 and source_video.status = 'active'
 and source_video.visibility_status = 'public'
 and (source_video.publish_at is null or source_video.publish_at <= now())
left join catalog.video_transcript_sentences sentence
  on sentence.video_id = source_video.video_id
 and sentence.sentence_index = wf.sentence_index
left join catalog.video_semantic_spans span
  on span.video_id = sentence.video_id
 and span.sentence_index = sentence.sentence_index
 and span.span_index = wf.token_index
where wf.user_id = sqlc.arg(user_id)::uuid
  and wf.is_favorited = true
  and wf.favorited_at is not null
  and (
    (wf.favorite_key_type = 'coarse_unit' and cu.id is not null)
    or
    (wf.favorite_key_type = 'video_token' and span.video_id is not null)
  )
  and (
    not sqlc.arg(has_cursor)::boolean
    or wf.favorited_at < sqlc.arg(cursor_favorited_at)::timestamptz
    or (
      wf.favorited_at = sqlc.arg(cursor_favorited_at)::timestamptz
      and wf.favorite_id > sqlc.arg(cursor_favorite_id)::uuid
    )
  )
order by wf.favorited_at desc, wf.favorite_id asc
limit sqlc.arg(limit_plus_one)::integer;
