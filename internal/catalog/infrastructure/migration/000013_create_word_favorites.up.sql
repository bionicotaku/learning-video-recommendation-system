create table if not exists catalog.word_favorites (
  favorite_id uuid primary key default gen_random_uuid(),
  user_id uuid not null
    references auth.users(id) on delete cascade,
  favorite_key_type text not null
    check (favorite_key_type in ('coarse_unit', 'video_token')),
  coarse_unit_id bigint,
  source text not null
    check (source in ('word_list', 'video_transcript')),
  video_id uuid,
  sentence_index integer,
  token_index integer,
  is_favorited boolean not null default false,
  favorited_at timestamptz,
  state_updated_at timestamptz not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  check (
    (
      favorite_key_type = 'coarse_unit'
      and coarse_unit_id is not null
    )
    or
    (
      favorite_key_type = 'video_token'
      and coarse_unit_id is null
      and video_id is not null
      and sentence_index is not null
      and token_index is not null
    )
  ),
  check (
    source = 'word_list'
    or
    (
      source = 'video_transcript'
      and video_id is not null
      and sentence_index is not null
      and token_index is not null
    )
  ),
  check (sentence_index is null or sentence_index >= 0),
  check (token_index is null or token_index >= 0),
  check (
    (
      is_favorited = true
      and favorited_at is not null
    )
    or
    (
      is_favorited = false
      and favorited_at is null
    )
  )
);

create unique index if not exists uq_word_favorites_coarse_unit
on catalog.word_favorites (user_id, coarse_unit_id)
where favorite_key_type = 'coarse_unit';

create unique index if not exists uq_word_favorites_video_token
on catalog.word_favorites (user_id, video_id, sentence_index, token_index)
where favorite_key_type = 'video_token';

create index if not exists idx_word_favorites_page
on catalog.word_favorites (user_id, favorited_at desc, favorite_id asc)
where is_favorited = true and favorited_at is not null;
