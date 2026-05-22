# API

`internal/api` is the external HTTP traffic entrypoint and API composition layer.

It owns HTTP transport concerns:

- trusted principal extraction
- request parsing and transport validation
- route registration
- response and error envelopes
- middleware for request id, recovery, timeout, body limits, and logging
- API-level orchestration across business usecases

The current mobile MVP does not implement CORS middleware. If a browser-based
client is introduced later, add a dedicated CORS middleware and allowlist
configuration then.

It does not own business tables, migrations, SQLC packages, or domain rules.
Business rules remain in `catalog`, `analytics`, `learningengine`,
`recommendation`, `semantic`, and `user`. API may define small facade ports and
transaction adapters when one HTTP endpoint must commit multiple module-owned
repositories together.

Current implemented endpoint group:

```text
POST /api/feed
GET /api/videos/{video_id}
GET /api/video-favorites
GET /api/video-history
GET /api/me
POST /api/videos/end-quiz
GET /api/unit-collections
GET /api/learning-targets/active-coarse-unit-ids
PUT /api/learning-targets/active-collection
PUT /api/videos/{video_id}/like
DELETE /api/videos/{video_id}/like
PUT /api/videos/{video_id}/favorite
DELETE /api/videos/{video_id}/favorite
POST /api/learning-interactions:batch
POST /api/quiz-attempts
POST /api/learning-units:mark-mastered
POST /api/video-watch-progress
GET /api/learning/unit-progress/mastered
GET /api/learning/unit-progress/unmastered
POST /api/feedback
```

`POST /api/feed` is a facade endpoint. It calls Recommendation to generate the
feed plan and audit/serving state, then batch-fills Catalog list preview
fields and `semantic.coarse_unit.label` text before returning the frontend
`FeedResponse`.
It does not expose recommendation rank, score, reason codes, selector mode, or
underfilled status. It also does not return playback detail, transcript URL,
global like/favorite counts, or current user like/favorite flags; those belong
to `GET /api/videos/{video_id}`.

Feed materialization is all-or-error: the facade does not silently drop plan
items or learning units after Recommendation has written audit/serving state.
Missing display data, incomplete evidence, or missing unit labels are treated as
backend consistency failures.

`GET /api/videos/{video_id}` reads the trusted principal as `user_id`, validates
the path UUID, and calls Catalog for one active/public/published video. It
returns playback URL, optional transcript URL, description, duration, view /
like / favorite counts, and current user like / favorite state. Missing
transcript metadata returns `transcript_url: null`; missing stats or user state
return zero counts and false flags. The endpoint is read-only and does not write
Analytics, Learning Engine, Recommendation, or Catalog interaction state.

`GET /api/video-favorites` and `GET /api/video-history` read the trusted
principal as `user_id`, parse optional `limit` and `cursor`, and call Catalog
read usecases for keyset-paginated video lists. Favorites return list preview
fields plus `favorited_at`; history returns list preview fields plus
`last_position_ms` and `last_watched_at`. Both endpoints filter to
active/public/published videos and do not return playback URLs, transcript URLs,
descriptions, like/favorite counts, or current user interaction flags.

`GET /api/me` reads the trusted principal as `user_id`, returns the User profile
cache plus precomputed global activity stats and an embedded seven-day activity
calendar, and may update the stored profile timezone when `X-Client-Timezone`
contains a valid IANA timezone. The activity calendar returns today plus the
previous six days in ascending date order and includes `current_streak_days`;
day rows do not include an `is_active` boolean. It does not aggregate Catalog,
Analytics, or Learning Engine tables at request time.

`POST /api/videos/end-quiz` is a read-only quiz lookup endpoint for the video
ending experience. The handler validates `video_id`, de-duplicates up to eight
`coarse_unit_ids`, validates optional `recommendation_run_id` and
`client_context`, then calls Catalog. It does not write quiz delivery/session
state and does not participate in Learning Engine progress updates; completed
answers still go through `POST /api/quiz-attempts`.

`GET /api/unit-collections` is owned by the `unitcollections` HTTP handler. It
reads the trusted principal as `user_id`, lists active Semantic unit collections
for target selection, and returns the current user's `active_collection` slug or
`null` from Learning Engine profile state.

`PUT /api/learning-targets/active-collection` is owned by the `learningtargets`
HTTP handler. It reads the trusted principal as `user_id`, validates
`collection_slug`, and opens one user-scoped transaction that switches the
Learning Engine collection target projection and updates User onboarding to
`collection_selected`. API does not pull collection members into memory or
bypass the owning module repositories. The endpoint is synchronous: `200 OK`
means the target projection and onboarding update are already committed; there
is no activation job or background switching state in the MVP.

`GET /api/learning-targets/active-coarse-unit-ids` is also owned by the
`learningtargets` HTTP handler. It reads the trusted principal as `user_id` and
returns the current Learning Engine target projection for fullscreen exposure
filtering. It returns `active_collection` from `learning.user_learning_profiles`
and `coarse_unit_ids` from `learning.user_unit_states` where `is_target=true`
and `status!='mastered'`. Missing active profile is a successful empty response.

`PUT/DELETE /api/videos/{video_id}/like` and
`PUT/DELETE /api/videos/{video_id}/favorite` are bodyless idempotent set/unset
endpoints. The handler reads `video_id` from the path, validates the trusted
principal, and calls Catalog. Like responses return only `video_id`,
`has_liked`, and `like_count`; favorite responses return only `video_id`,
`has_favorited`, and `favorite_count`.

`GET /api/videos/{video_id}` initializes action rail display with both global
counts and current user state. Video Favorites / Video History list endpoints
only provide navigation previews. Click writes still use the single-purpose
Video Interactions endpoints above.

The learning-event endpoints return only raw Analytics acceptance results.
Learning Engine normalization is attempted synchronously as best effort and is
not exposed as the HTTP success boundary.

`POST /api/learning-interactions:batch` accepts only exposure and lookup raw
interactions. Self-mark mastered is intentionally a separate endpoint so it can
use the dedicated Analytics writer and `NormalizeSelfMarkMasteredByID`
normalizer path. Before writing the raw fact, self-mark mastered requires an
existing `learning.user_unit_states` row for the current user and
`coarse_unit_id`; existing inactive or already mastered states are still
accepted and reduced to terminal mastered with `is_target=false`.

`POST /api/video-watch-progress` calls the Catalog `RecordVideoWatchProgress`
usecase. It returns only `{ "accepted": true }` after the watch session ledger
and Catalog projections have been updated in one backend transaction.

`GET /api/learning/unit-progress/mastered` and
`GET /api/learning/unit-progress/unmastered` call the Learning Engine reducer
read usecase. The handler reads only `limit` and `cursor` from query params,
uses the trusted principal as `user_id`, and returns the frontend display
contract defined in `docs/API/Unit-Progress-API-MVP设计.md`.

`POST /api/feedback` reads the trusted principal as `user_id`, accepts
`multipart/form-data` with one frontend-owned JSON object `payload` and up to
five JPEG `images`, and calls User `SubmitFeedback`. The route has a dedicated
5 MiB request body limit; other API routes keep the default 1 MiB body limit.
The endpoint writes `app_user.feedback_submissions` and
`app_user.feedback_images` atomically through the User module and stores image
bytes as Postgres `bytea`, not base64 JSON.

`cmd/server` reads principal configuration from environment variables. In normal
mode it expects GCP API Gateway to validate the client JWT and forward
`X-Apigateway-Api-Userinfo`; the API auth middleware decodes that userinfo
payload and uses the `sub` claim as `principal.UserID`.

When `DEV_MODE=true`, the same gateway header is still preferred. If it is
absent, the middleware may fall back to `Authorization: Bearer <JWT>` for trusted
frontend direct-connect testing. That fallback only decodes the JWT payload; it
does not verify signatures and must not be used as a production trust boundary.
The module never trusts `user_id` from the body or query string.

Relevant environment variables are documented in `.env.example`. `DATABASE_URL`
and `PUBLIC_ASSET_BASE_URL` are required. `DEV_MODE` defaults to `false`, and
`API_GATEWAY_USERINFO_HEADER` defaults to `X-Apigateway-Api-Userinfo`.

`cmd/server` also requires `PUBLIC_ASSET_BASE_URL` for feed media URL assembly.
Catalog paths that are already absolute `http://` or `https://` URLs pass
through unchanged; relative media and cover paths are joined against this base
URL. Current catalog ingest writes absolute GCS mp4 and cover URLs, so those
values pass through without using this prefix.

Transport errors use the shared JSON error envelope. Decode, field validation,
and known business validation failures map to `400 invalid_request`; missing
principal maps to `401 unauthorized`; feedback request body overflow maps to
`413 payload_too_large`; known owner errors can map to `404 not_found`, `409
conflict`, or `422 unprocessable_entity`; timeouts or canceled contexts map to a
request-id-bearing `503 service_unavailable`; unknown usecase failures map to
`500 internal_error`.
