# Semantic Module

`internal/semantic` owns system learning unit collection metadata.

Current implementation:

- `semantic.unit_collections` stores available system collections such as
  wordbooks. `slug` is the API identifier and must be lowercase
  `^[a-z0-9][a-z0-9-]{0,80}$`.
- `semantic.unit_collection_members` stores collection membership by `coarse_unit_id`.
- `ListUnitCollectionsUsecase` returns active collections for API display.
- `scripts/unit_collection_ingest` imports local standard JSON wordbooks. It
  stores `word_unit_count` as the distinct matched `headWord` count and
  `coarse_unit_count` as the final collection member count.

`semantic.unit_collections.internal_description` and `semantic.unit_collections.source_payload`
are internal/admin-only fields. Public collection list reads do not select them.

Boundary:

- Semantic owns collection definitions and membership reads.
- Learning Engine owns per-user active collection and target projection.
- Recommendation does not read collection tables directly.
