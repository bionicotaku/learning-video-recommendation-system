-- name: UpsertFeedbackSubmission :one
insert into app_user.feedback_submissions (
  id,
  user_id,
  client_feedback_id,
  payload
) values (
  sqlc.arg(id),
  sqlc.arg(user_id),
  sqlc.narg(client_feedback_id),
  sqlc.arg(payload)
)
on conflict (user_id, client_feedback_id) do update
set client_feedback_id = excluded.client_feedback_id
returning id, created_at, (xmax = 0) as inserted;

-- name: InsertFeedbackImage :exec
insert into app_user.feedback_images (
  id,
  submission_id,
  sort_order,
  content_type,
  size_bytes,
  sha256,
  width,
  height,
  image_data
) values (
  sqlc.arg(id),
  sqlc.arg(submission_id),
  sqlc.arg(sort_order),
  sqlc.arg(content_type),
  sqlc.arg(size_bytes),
  sqlc.arg(sha256),
  sqlc.arg(width),
  sqlc.arg(height),
  sqlc.arg(image_data)
);

-- name: CountFeedbackImages :one
select count(*)::integer
from app_user.feedback_images
where submission_id = sqlc.arg(submission_id);
