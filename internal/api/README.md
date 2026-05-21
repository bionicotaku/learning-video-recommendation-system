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

It does not own business tables, migrations, SQLC packages, repositories, or
domain rules. Business rules remain in `catalog`, `analytics`,
`learningengine`, `recommendation`, `semantic`, and `user`.

Current implemented endpoint group:

```text
POST /api/feed
GET /api/me
GET /api/me/activity-calendar
POST /api/videos/end-quiz
GET /api/unit-collections
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
```

`POST /api/feed` is a facade endpoint. It calls Recommendation to generate the
feed plan and audit/serving state, then batch-fills Catalog video display
fields, engagement stats, and `semantic.coarse_unit.label` text before returning
the frontend `FeedResponse`. It does not expose recommendation rank, score,
reason codes, selector mode, or underfilled status.

Feed materialization is all-or-error: the facade does not silently drop plan
items or learning units after Recommendation has written audit/serving state.
Missing display data, incomplete evidence, missing unit labels, or invalid media
URLs are treated as backend consistency failures.

`GET /api/me` reads the trusted principal as `user_id`, returns the User profile
cache plus precomputed global activity stats, and may update the stored profile
timezone when `X-Client-Timezone` contains a valid IANA timezone. It does not
aggregate Catalog, Analytics, or Learning Engine tables at request time.

`GET /api/me/activity-calendar` returns today plus the previous six days in
ascending date order. It uses a valid `X-Client-Timezone` if provided, otherwise
falls back to the stored profile timezone and then UTC. This endpoint never
updates the stored profile timezone.

`POST /api/videos/end-quiz` is a read-only quiz lookup endpoint for the video
ending experience. The handler validates `video_id`, de-duplicates up to eight
`coarse_unit_ids`, validates optional `recommendation_run_id` and
`client_context`, then calls Catalog. It does not write quiz delivery/session
state and does not participate in Learning Engine progress updates; completed
answers still go through `POST /api/quiz-attempts`.

`GET /api/unit-collections` lists active Semantic unit collections for target
selection. `PUT /api/learning-targets/active-collection` reads the trusted
principal as `user_id`, validates `collection_slug`, and calls Learning Engine
to switch the user's collection target projection in one transaction. API does
not pull collection members into memory or write `learning.*` directly.

`PUT/DELETE /api/videos/{video_id}/like` and
`PUT/DELETE /api/videos/{video_id}/favorite` are bodyless idempotent set/unset
endpoints. The handler reads `video_id` from the path, validates the trusted
principal, and calls Catalog. Like responses return only `video_id`,
`has_liked`, and `like_count`; favorite responses return only `video_id`,
`has_favorited`, and `favorite_count`.

The learning-event endpoints return only raw Analytics acceptance results.
Learning Engine normalization is attempted synchronously as best effort and is
not exposed as the HTTP success boundary.

`POST /api/learning-interactions:batch` accepts only exposure and lookup raw
interactions. Self-mark mastered is intentionally a separate endpoint so it can
use the dedicated Analytics writer and `NormalizeSelfMarkMasteredByID`
normalizer path.

`POST /api/video-watch-progress` calls the Catalog `RecordVideoWatchProgress`
usecase. It returns only `{ "accepted": true }` after the watch session ledger
and Catalog projections have been updated in one backend transaction.

`GET /api/learning/unit-progress/mastered` and
`GET /api/learning/unit-progress/unmastered` call the Learning Engine reducer
read usecase. The handler reads only `limit` and `cursor` from query params,
uses the trusted principal as `user_id`, and returns the frontend display
contract defined in `docs/API/Unit-Progress-API-MVP设计.md`.

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
principal maps to `401 unauthorized`; known owner errors can map to `404
not_found`, `409 conflict`, or `422 unprocessable_entity`; timeouts or canceled
contexts map to a request-id-bearing `503 service_unavailable`; unknown usecase
failures map to `500 internal_error`.
