-- name: GetUserSchedulerSettings :one
select *
from learning.user_scheduler_settings
where user_id = sqlc.arg(user_id)
limit 1;
