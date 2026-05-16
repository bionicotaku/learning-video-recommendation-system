# Platform

`internal/platform` contains cross-module technical primitives that do not belong to a business owner.

This layer is intentionally narrow:

- It may depend on the Go standard library and third-party technical libraries.
- It must not import `internal/api`, `internal/catalog`, `internal/analytics`, `internal/learningengine`, or `internal/recommendation`.
- It must not contain business DTOs, domain models, policies, rules, repositories, SQL queries, migrations, or cross-owner data access helpers.
- Packages under this directory must be named by a specific technical responsibility, not by generic names such as `utils`, `common`, or `helper`.

## Current Packages

- `postgres/pgtime`: shared `time.Time <-> pgtype.Timestamptz` mapping with UTC normalization.
- `postgres/pguuid`: shared `string <-> pgtype.UUID` mapping.
- `postgres/pgtext`: shared `string <-> pgtype.Text` mapping where empty strings map to invalid nullable text.
- `postgres/pgnumeric`: shared `float64 <-> pgtype.Numeric` mapping without domain rounding.
- `postgres/pgtest`: shared embedded Postgres test harness, template database cloning, and ordered schema plan execution.

Business modules still keep local persistence mapper functions. Those local functions preserve module vocabulary and delegate pure Postgres type conversion to this platform layer.

`postgres/pgtest` is only a technical test primitive. It must not contain business seed helpers, module schema ownership decisions, repository constructors, or cross-owner data access shortcuts. Module fixtures and E2E harnesses define their own schema plans and seed semantics.
