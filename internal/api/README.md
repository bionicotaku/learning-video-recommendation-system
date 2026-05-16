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
`learningengine`, and `recommendation`.

Current implemented endpoint group:

```text
POST /api/feed
POST /api/learning-interactions:batch
POST /api/quiz-attempts
POST /api/learning-units:mark-mastered
POST /api/video-watch-progress
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

`cmd/server` requires `API_TRUSTED_USER_ID_HEADER`. The header must be injected
by a trusted upstream gateway or runtime that strips client-supplied identity
headers before forwarding the request. This module does not implement JWT
verification and never trusts `user_id` from the body or query string.

`cmd/server` also requires `PUBLIC_ASSET_BASE_URL` for feed media URL assembly.
Catalog paths that are already absolute `http://` or `https://` URLs pass
through unchanged; relative HLS and cover paths are joined against this base URL.

Transport errors use the shared JSON error envelope. Decode, field validation,
and known business validation failures map to `400 invalid_request`; missing
principal maps to `401 unauthorized`; known owner errors can map to `404
not_found`, `409 conflict`, or `422 unprocessable_entity`; timeouts or canceled
contexts map to a request-id-bearing `503 service_unavailable`; unknown usecase
failures map to `500 internal_error`.
