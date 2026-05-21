# Semantic Module

`internal/semantic` owns system learning unit collection metadata.

Current implementation:

- `semantic.unit_collections` stores available system collections such as wordbooks.
- `semantic.unit_collection_members` stores collection membership by `coarse_unit_id`.
- `ListUnitCollectionsUsecase` returns active collections for API display.

Boundary:

- Semantic owns collection definitions and membership reads.
- Learning Engine owns per-user active collection and target projection.
- Recommendation does not read collection tables directly.
