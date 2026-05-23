# User Module

`internal/user` owns application-level user profile data and activity
projections. Supabase Auth remains the identity source; this module stores a
profile cache and precomputed counters used by API responses.
It also owns low-volume user support feedback submissions for the current MVP.

## Owned Tables

The module owns the `app_user` schema:

- `app_user.user_profiles`
- `app_user.user_activity_stats`
- `app_user.user_daily_activity_stats`
- `app_user.feedback_submissions`
- `app_user.feedback_images`

`auth.users.email` is the authoritative email source. `user_profiles.email` is
only a cache kept in sync by Supabase Auth triggers. Business modules must not
write `app_user.*` SQL directly; they use the User application ports when they
need to update projections inside an existing transaction.

## Main Usecases

- `GetMe`: returns profile fields, global activity stats, and the embedded
  seven-day activity calendar. If the profile row is missing, it repairs it
  from `auth.users`. A valid `X-Client-Timezone` value updates the profile
  timezone before the activity calendar is computed.
- `UpdateMeProfile`: updates current-user editable profile fields
  (`display_name`, `birth_date`, `gender`, `education_stage`, `timezone`) after
  validating display name, date range, enums, and IANA timezone values. Missing
  profile rows are repaired from `auth.users` before applying the patch.
- `UpdateOnboardingStatus`: updates the profile onboarding state after flows
  such as learning target collection selection.
- `ActivityStatsRecorder`: transaction-aware projection writer for watch time,
  quiz attempts, started units, and daily learning interactions.
- `SubmitFeedback`: stores one current-user feedback submission, an arbitrary
  JSON object payload, and up to five validated JPEG images in one transaction.
  It supports `client_feedback_id` idempotency for frontend retries.

## Cross-Module Boundary

Catalog, Analytics, and Learning Engine can receive an `ActivityStatsRecorder`
bound to their current transaction. This keeps their domain writes and User
projection updates atomic without transferring ownership of `app_user.*` tables.

Current integrations:

- Catalog watch progress adds positive active watch deltas to global and daily
  watch stats.
- Analytics increments quiz attempts only for newly inserted quiz events, and
  increments daily learning interactions only for newly inserted raw exposure or
  lookup events.
- Learning Engine increments `started_unit_count` once when a unit crosses from
  no progress to positive progress.

## HTTP Exposure

The HTTP handlers live under `internal/api`; User only provides usecases. Current
API endpoints:

- `GET /api/me`
- `PATCH /api/me/profile`
- `POST /api/feedback`
