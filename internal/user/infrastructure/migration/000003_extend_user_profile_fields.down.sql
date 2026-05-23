alter table app_user.user_profiles
  drop column if exists ip_region,
  drop constraint if exists user_profiles_education_stage_check,
  drop column if exists education_stage,
  drop constraint if exists user_profiles_gender_check,
  drop column if exists gender,
  drop column if exists birth_date,
  drop constraint if exists user_profiles_display_name_non_empty_check,
  alter column display_name drop not null;

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
