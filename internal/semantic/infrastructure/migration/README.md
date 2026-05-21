# Semantic Migrations

Semantic owns system learning unit collection metadata.

Current owner tables:

- `semantic.unit_collections`
- `semantic.unit_collection_members`

`semantic.unit_collections.slug` is the API-facing identifier. It is stored only
as lowercase `^[a-z0-9][a-z0-9-]{0,80}$`; ingest must canonicalize source
filenames before writing.

`semantic.coarse_unit` already exists in the shared database and is referenced by membership rows.
