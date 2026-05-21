create schema if not exists app_user;

alter table auth.users
add column if not exists email text;

alter table auth.users
add column if not exists email_confirmed_at timestamptz;

create table if not exists app_user.user_profiles (
  user_id uuid primary key references auth.users(id) on delete cascade,

  email text,
  email_confirmed_at timestamptz,

  display_name text,
  avatar_url text,

  locale text not null default 'zh-CN',
  timezone text,

  onboarding_status text not null default 'new'
    check (onboarding_status in ('new', 'collection_selected', 'completed')),

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_user_profiles_email
on app_user.user_profiles (email)
where email is not null;

create table if not exists app_user.user_activity_stats (
  user_id uuid primary key references auth.users(id) on delete cascade,

  total_watch_ms bigint not null default 0,
  quiz_attempt_count bigint not null default 0,
  started_unit_count bigint not null default 0,

  updated_at timestamptz not null default now(),

  check (total_watch_ms >= 0),
  check (quiz_attempt_count >= 0),
  check (started_unit_count >= 0)
);

create table if not exists app_user.user_daily_activity_stats (
  user_id uuid not null references auth.users(id) on delete cascade,

  local_date date not null,
  timezone text not null,

  watch_ms bigint not null default 0,
  quiz_attempt_count bigint not null default 0,
  learning_interaction_count bigint not null default 0,

  first_activity_at timestamptz,
  last_activity_at timestamptz,
  updated_at timestamptz not null default now(),

  primary key (user_id, local_date),

  check (watch_ms >= 0),
  check (quiz_attempt_count >= 0),
  check (learning_interaction_count >= 0)
);

create index if not exists idx_user_daily_activity_stats_user_date_desc
on app_user.user_daily_activity_stats (user_id, local_date desc);

create or replace function app_user.handle_auth_user_created()
returns trigger
language plpgsql
security definer
set search_path = app_user, auth, public
as $$
begin
  insert into app_user.user_profiles (
    user_id,
    email,
    email_confirmed_at,
    display_name,
    locale,
    onboarding_status
  )
  values (
    new.id,
    new.email,
    new.email_confirmed_at,
    nullif(split_part(coalesce(new.email, ''), '@', 1), ''),
    'zh-CN',
    'new'
  )
  on conflict (user_id) do nothing;

  insert into app_user.user_activity_stats (user_id)
  values (new.id)
  on conflict (user_id) do nothing;

  return new;
end;
$$;

drop trigger if exists on_auth_user_created on auth.users;

create trigger on_auth_user_created
after insert on auth.users
for each row execute function app_user.handle_auth_user_created();

create or replace function app_user.handle_auth_user_email_updated()
returns trigger
language plpgsql
security definer
set search_path = app_user, auth, public
as $$
begin
  update app_user.user_profiles
  set
    email = new.email,
    email_confirmed_at = new.email_confirmed_at,
    updated_at = now()
  where user_id = new.id;

  return new;
end;
$$;

drop trigger if exists on_auth_user_email_updated on auth.users;

create trigger on_auth_user_email_updated
after update of email, email_confirmed_at on auth.users
for each row
when (
  old.email is distinct from new.email
  or old.email_confirmed_at is distinct from new.email_confirmed_at
)
execute function app_user.handle_auth_user_email_updated();
