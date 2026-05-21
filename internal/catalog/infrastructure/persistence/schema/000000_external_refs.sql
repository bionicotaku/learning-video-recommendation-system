create schema if not exists auth;
create schema if not exists semantic;
create schema if not exists analytics;

create table if not exists auth.users (
  id uuid primary key,
  email text,
  email_confirmed_at timestamptz
);

create sequence if not exists semantic.coarse_unit_id_seq;

create table if not exists semantic.coarse_unit (
  id bigint primary key default nextval('semantic.coarse_unit_id_seq'::regclass),
  label text not null default '',
  status text not null default 'active'
);
