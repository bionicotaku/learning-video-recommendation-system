alter table learning.unit_learning_events
  drop constraint if exists chk_unit_learning_events_set_mastered_event_type;

alter table learning.unit_learning_events
  drop constraint if exists chk_unit_learning_events_progress_quality,
  add constraint chk_unit_learning_events_progress_quality
    check (
      (reducer_effect = 'affects_progress' and progress_quality between 0 and 5)
      or
      (reducer_effect = 'observe_only' and progress_quality is null)
    );

alter table learning.unit_learning_events
  drop constraint if exists chk_unit_learning_events_reducer_effect,
  drop constraint if exists unit_learning_events_reducer_effect_check,
  add constraint unit_learning_events_reducer_effect_check
    check (reducer_effect in ('observe_only', 'affects_progress'));
