drop trigger if exists on_auth_user_email_updated on auth.users;
drop function if exists app_user.handle_auth_user_email_updated();

drop trigger if exists on_auth_user_created on auth.users;
drop function if exists app_user.handle_auth_user_created();

drop table if exists app_user.user_daily_activity_stats;
drop table if exists app_user.user_activity_stats;
drop table if exists app_user.user_profiles;
drop schema if exists app_user;
