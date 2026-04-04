create schema if not exists auth;
create table if not exists auth.users (
  id uuid primary key
);

create schema if not exists catalog;
create table if not exists catalog.videos (
  video_id uuid primary key
);

create schema if not exists semantic;
create table if not exists semantic.coarse_unit (
  id bigint primary key,
  kind text not null,
  label text not null,
  pos text,
  english_def text,
  chinese_def text
);
