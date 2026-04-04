alter table learning.user_unit_states
  drop column if exists recent_correctness_window,
  drop column if exists recent_quality_window;
