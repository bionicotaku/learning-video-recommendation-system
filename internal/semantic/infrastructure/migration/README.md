# Semantic Migrations

Semantic owns durable base vocabulary data and system learning unit collection metadata.

Current baseline files:

- `000001_baseline.up.sql`
- `000001_baseline.down.sql`

Current owner tables:

- `semantic.fine_unit`
- `semantic.coarse_unit`
- `semantic.unit_collections`
- `semantic.unit_collection_members`

`semantic.unit_collections.slug` is the API-facing identifier. It is stored only
as lowercase `^[a-z0-9][a-z0-9-]{0,80}$`; ingest must canonicalize source
filenames before writing.

`semantic.coarse_unit` and `semantic.fine_unit` contain preserved base data. Do not
drop or truncate this schema during baseline cleanup; if migration history is
squashed on an existing database, update only `semantic_schema_migrations`.
