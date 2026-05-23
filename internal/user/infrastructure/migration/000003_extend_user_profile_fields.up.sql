update app_user.user_profiles
set
  display_name = coalesce(
    nullif(btrim(display_name), ''),
    nullif(split_part(coalesce(email, ''), '@', 1), ''),
    'user'
  ),
  updated_at = now()
where display_name is null or btrim(display_name) = '';

alter table app_user.user_profiles
  alter column display_name set not null,
  add constraint user_profiles_display_name_non_empty_check
    check (length(btrim(display_name)) > 0),
  add column birth_date date,
  add column gender text,
  add constraint user_profiles_gender_check
    check (gender is null or gender in ('male', 'female', 'other', 'prefer_not_to_say')),
  add column education_stage text,
  add constraint user_profiles_education_stage_check
    check (education_stage is null or education_stage in (
      'middle_school',
      'high_school',
      'undergraduate',
      'graduate',
      'phd',
      'working',
      'other'
    )),
  add column ip_region text;

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
