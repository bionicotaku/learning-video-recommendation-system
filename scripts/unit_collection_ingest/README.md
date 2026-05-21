# Unit Collection Ingest

This script imports standard JSON wordbooks into `semantic.unit_collections`
and `semantic.unit_collection_members`.

Input contract:

- The input directory contains top-level `*.json` files.
- Each file is a top-level JSON array.
- Each entry must contain integer `wordRank` and non-empty string `headWord`.
- Other entry fields are preserved in `semantic.unit_collections.source_payload`.

Generated files are written under the input directory:

```text
_unit_collection_ingest/
  collection_metadata.json
  matches/
    <slug>.matches.json
```

`slug` is the lowercased source filename without the `.json` suffix, and it
must match `^[a-z0-9][a-z0-9-]{0,80}$`. For example,
`new-oriental-GRE.json` writes `new-oriental-gre`. The metadata file is
generated with default `name`, `description`, and `internal_description` values;
manual edits are preserved on later runs, and only missing slugs are added.

## Commands

```bash
.venv/bin/python -m scripts.unit_collection_ingest.main match \
  --input-dir "/Users/evan/Downloads/词书"

.venv/bin/python -m scripts.unit_collection_ingest.main write \
  --input-dir "/Users/evan/Downloads/词书"

.venv/bin/python -m scripts.unit_collection_ingest.main all \
  --input-dir "/Users/evan/Downloads/词书"
```

`match` parses all source files, strictly matches `headWord` to
`semantic.coarse_unit.label`, and writes one match file per source. If a match
file already exists, the first stage skips that source output directly.

`write` verifies each match file still belongs to the current source file by
`source_sha256`, then upserts `semantic.unit_collections` by `slug` and replaces
that collection's `semantic.unit_collection_members` set.

## Count Semantics

- `word_unit_count` is the number of distinct source `headWord` values that have
  at least one strict `headWord -> coarse_unit` match.
- `coarse_unit_count` is the final member row count for that collection.
- `unit_collection_members` is unique by `(collection_id, coarse_unit_id)`, so if
  duplicate coarse units appear through multiple entries, the member row keeps
  the earliest `wordRank` as `sort_order`.

The script also prints the raw strict match count during the write stage so
duplicate collapse is visible.
