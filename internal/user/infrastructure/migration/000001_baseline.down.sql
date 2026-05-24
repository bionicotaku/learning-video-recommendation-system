drop trigger if exists on_auth_user_email_updated on auth.users;
drop function if exists app_user.handle_auth_user_email_updated();

drop trigger if exists on_auth_user_created on auth.users;
drop function if exists app_user.handle_auth_user_created();

drop schema if exists app_user cascade;
