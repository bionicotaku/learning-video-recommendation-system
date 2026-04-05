alter table if exists learning.user_unit_states
  drop column if exists last_recommended_at;

drop table if exists learning.scheduler_run_items;
drop table if exists learning.scheduler_runs;
drop table if exists learning.user_scheduler_settings;
