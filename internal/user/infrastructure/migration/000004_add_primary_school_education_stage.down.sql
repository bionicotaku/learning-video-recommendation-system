update app_user.user_profiles
set
  education_stage = null,
  updated_at = now()
where education_stage = 'primary_school';

alter table app_user.user_profiles
  drop constraint if exists user_profiles_education_stage_check;

alter table app_user.user_profiles
  add constraint user_profiles_education_stage_check
    check (education_stage is null or education_stage in (
      'middle_school',
      'high_school',
      'undergraduate',
      'graduate',
      'phd',
      'working',
      'other'
    ));
