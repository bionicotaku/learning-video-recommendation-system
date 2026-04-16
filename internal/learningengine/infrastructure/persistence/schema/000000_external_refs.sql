create schema if not exists auth;
create schema if not exists semantic;
create schema if not exists catalog;

create table if not exists auth.users (
  id uuid primary key
);

create table if not exists semantic.coarse_unit (
  id bigint primary key
);

create table if not exists catalog.videos (
  video_id uuid primary key
);
