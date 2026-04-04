# Scheduler Module

`internal/recommendation/scheduler` is the learning-content scheduling subsystem inside the recommendation module.

## Responsibilities

- Maintain `user x coarse_unit` scheduling state.
- Record normalized learning events.
- Generate recommendation batches for downstream recommendation stages.
- Support replay-based state rebuild and audit snapshots.

## Non-Responsibilities

- It is not a frontend API.
- It does not define HTTP handlers, controllers, or transport DTOs.
- It does not implement video recall or final task assembly for the user-facing product.
- It does not access Supabase through the HTTP SDK for core scheduling flows.
