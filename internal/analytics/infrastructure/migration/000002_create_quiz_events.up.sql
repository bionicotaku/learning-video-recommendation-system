create table if not exists analytics.quiz_events (
  event_id uuid primary key default gen_random_uuid(),

  client_event_id text not null,

  user_id uuid not null
    references auth.users(id) on delete cascade,

  question_id uuid not null
    references catalog.questions(question_id) on delete restrict,

  coarse_unit_id bigint not null
    references semantic.coarse_unit(id) on delete restrict,

  video_id uuid
    references catalog.videos(video_id) on delete set null,

  recommendation_run_id uuid,

  trigger_type text not null
    check (trigger_type in (
      'video_end',
      'lookup_practice',
      'feed_review',
      'mid_video',
      'manual'
    )),

  selected_option_ids text[] not null,
  selection_interval_ms integer[] not null,

  is_first_try_correct boolean not null,
  total_elapsed_ms integer not null,

  shown_at timestamptz not null,
  completed_at timestamptz not null,

  created_at timestamptz not null default now(),

  check (cardinality(selected_option_ids) >= 1),
  check (cardinality(selected_option_ids) = cardinality(selection_interval_ms)),
  check (selected_option_ids[cardinality(selected_option_ids)] = 'correct'),
  check (is_first_try_correct = (selected_option_ids[1] = 'correct')),
  check (total_elapsed_ms >= 0),
  check (completed_at >= shown_at)
);

create unique index if not exists uq_quiz_events_user_client_event
on analytics.quiz_events (user_id, client_event_id);

create index if not exists idx_quiz_events_user_completed_at
on analytics.quiz_events (user_id, completed_at desc);

create index if not exists idx_quiz_events_question_completed_at
on analytics.quiz_events (question_id, completed_at desc);

create index if not exists idx_quiz_events_unit_completed_at
on analytics.quiz_events (coarse_unit_id, completed_at desc);

create index if not exists idx_quiz_events_video_completed_at
on analytics.quiz_events (video_id, completed_at desc)
where video_id is not null;
