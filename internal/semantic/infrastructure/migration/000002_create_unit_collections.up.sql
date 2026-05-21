create table if not exists semantic.unit_collections (
  collection_id uuid primary key default gen_random_uuid(),
  slug text not null unique,
  name text not null,
  description text,
  category text not null default 'wordbook',
  status text not null default 'active'
    check (status in ('active', 'inactive')),
  coarse_unit_count integer not null default 0
    check (coarse_unit_count >= 0),
  word_unit_count integer not null default 0
    check (word_unit_count >= 0),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  internal_description text,
  source_payload jsonb
);

create table if not exists semantic.unit_collection_members (
  collection_id uuid not null references semantic.unit_collections(collection_id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  sort_order integer not null,
  target_priority numeric(8,4) not null default 0,
  created_at timestamptz not null default now(),

  primary key (collection_id, coarse_unit_id)
);

create index if not exists idx_unit_collection_members_collection_order
on semantic.unit_collection_members (collection_id, sort_order, coarse_unit_id);

create index if not exists idx_unit_collection_members_unit
on semantic.unit_collection_members (coarse_unit_id, collection_id);
