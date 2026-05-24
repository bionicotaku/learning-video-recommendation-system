create schema if not exists app_user;

create table if not exists app_user.user_profiles (
  user_id uuid primary key references auth.users(id) on delete cascade,

  email text,
  email_confirmed_at timestamptz,

  display_name text not null
    constraint user_profiles_display_name_non_empty_check
    check (length(btrim(display_name)) > 0),
  avatar_url text,

  locale text not null default 'zh-CN',
  timezone text,

  onboarding_status text not null default 'new'
    check (onboarding_status in ('new', 'collection_selected', 'completed')),

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  birth_date date,
  gender text
    constraint user_profiles_gender_check
    check (gender is null or gender in ('male', 'female', 'other', 'prefer_not_to_say')),
  education_stage text
    constraint user_profiles_education_stage_check
    check (education_stage is null or education_stage in (
      'primary_school',
      'middle_school',
      'high_school',
      'undergraduate',
      'graduate',
      'phd',
      'working',
      'other'
    )),
  ip_region text
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

create table if not exists app_user.feedback_submissions (
  id uuid primary key,
  user_id uuid not null references auth.users(id) on delete cascade,
  client_feedback_id uuid,
  payload jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),

  unique (user_id, client_feedback_id),
  check (jsonb_typeof(payload) = 'object')
);

create table if not exists app_user.feedback_images (
  id uuid primary key,
  submission_id uuid not null references app_user.feedback_submissions(id) on delete cascade,
  sort_order integer not null,
  content_type text not null,
  size_bytes integer not null,
  sha256 text not null,
  width integer not null,
  height integer not null,
  image_data bytea not null,
  created_at timestamptz not null default now(),

  unique (submission_id, sort_order),
  check (sort_order between 1 and 5),
  check (content_type = 'image/jpeg'),
  check (size_bytes > 0),
  check (width > 0),
  check (height > 0)
);

create index if not exists idx_feedback_submissions_user_created_at_desc
on app_user.feedback_submissions (user_id, created_at desc);

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
    coalesce(nullif(split_part(coalesce(new.email, ''), '@', 1), ''), 'user'),
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
