create schema if not exists auth;
create schema if not exists semantic;
create schema if not exists catalog;

create table if not exists auth.users (
  id uuid primary key,
  email text,
  email_confirmed_at timestamptz
);

create sequence if not exists semantic.coarse_unit_id_seq;

create table if not exists semantic.coarse_unit (
  id bigint primary key default nextval('semantic.coarse_unit_id_seq'::regclass),
  kind text not null,
  label text not null,
  lang text not null default 'en',
  pos text,
  english_def text,
  chinese_def text,
  chinese_criteria text,
  chinese_label text,
  english_label text,
  pattern jsonb,
  status text not null default 'active',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  version integer not null default 1,
  fine_unit_ids bigint[] not null,
  original_defs text[] not null
);

create table if not exists semantic.unit_collections (
  collection_id uuid primary key,
  slug text not null unique
    constraint unit_collections_slug_canonical_check
    check (slug = lower(slug) and slug ~ '^[a-z0-9][a-z0-9-]{0,80}$'),
  name text not null default '',
  status text not null default 'active',
  coarse_unit_count integer not null default 0
);

create table if not exists semantic.unit_collection_members (
  collection_id uuid not null references semantic.unit_collections(collection_id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  sort_order integer not null,
  target_priority numeric(8,4) not null default 0,
  created_at timestamptz not null default now(),
  primary key (collection_id, coarse_unit_id)
);

create table if not exists catalog.videos (
  video_id uuid primary key
);
