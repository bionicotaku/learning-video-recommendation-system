create table if not exists app_user.feedback_submissions (
  id uuid primary key,
  user_id uuid not null references auth.users(id) on delete cascade,
  client_feedback_id uuid,
  payload jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),

  unique (user_id, client_feedback_id),
  check (jsonb_typeof(payload) = 'object')
);

create table if not exists app_user.feedback_images (
  id uuid primary key,
  submission_id uuid not null references app_user.feedback_submissions(id) on delete cascade,
  sort_order integer not null,
  content_type text not null,
  size_bytes integer not null,
  sha256 text not null,
  width integer not null,
  height integer not null,
  image_data bytea not null,
  created_at timestamptz not null default now(),

  unique (submission_id, sort_order),
  check (sort_order between 1 and 5),
  check (content_type = 'image/jpeg'),
  check (size_bytes > 0),
  check (width > 0),
  check (height > 0)
);

create index if not exists idx_feedback_submissions_user_created_at_desc
on app_user.feedback_submissions (user_id, created_at desc);
