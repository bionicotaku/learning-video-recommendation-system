alter table learning.unit_learning_events
  drop constraint if exists unit_learning_events_reducer_effect_check,
  drop constraint if exists chk_unit_learning_events_reducer_effect,
  add constraint chk_unit_learning_events_reducer_effect
    check (reducer_effect in ('observe_only', 'affects_progress', 'set_mastered'));

alter table learning.unit_learning_events
  drop constraint if exists chk_unit_learning_events_progress_quality,
  add constraint chk_unit_learning_events_progress_quality
    check (
      (reducer_effect = 'affects_progress' and progress_quality between 0 and 5)
      or
      (reducer_effect = 'observe_only' and progress_quality is null)
      or
      (reducer_effect = 'set_mastered' and progress_quality is null)
    );

alter table learning.unit_learning_events
  drop constraint if exists chk_unit_learning_events_set_mastered_event_type,
  add constraint chk_unit_learning_events_set_mastered_event_type
    check (
      reducer_effect <> 'set_mastered'
      or event_type = 'self_mark_mastered'
    );
