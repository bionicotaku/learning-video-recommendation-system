create schema if not exists semantic;

create extension if not exists pgcrypto;
create extension if not exists pg_trgm;

create sequence if not exists semantic.fine_unit_id_seq;

create table if not exists semantic.fine_unit (
  id bigint primary key default nextval('semantic.fine_unit_id_seq'::regclass),
  kind text not null check (kind in ('word_sense', 'phrase_sense', 'grammar_rule')),
  label text not null,
  lang text not null default 'en',
  pos char(1),
  def text,
  pattern jsonb,
  meta jsonb not null default '{}'::jsonb,
  status text not null default 'active',
  created_at timestamptz default now(),
  updated_at timestamptz default now(),
  external_key text
);

alter sequence semantic.fine_unit_id_seq owned by semantic.fine_unit.id;

create sequence if not exists semantic.coarse_unit_id_seq;

create table if not exists semantic.coarse_unit (
  id bigint primary key default nextval('semantic.coarse_unit_id_seq'::regclass),
  kind text not null check (kind in ('word', 'phrase', 'grammar')),
  label text not null,
  lang text not null default 'en',
  pos text,
  english_def text,
  chinese_def text,
  chinese_criteria text,
  chinese_label text,
  english_label text,
  pattern jsonb,
  status text not null default 'active',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  version integer not null default 1,
  fine_unit_ids bigint[] not null,
  original_defs text[] not null
);

alter sequence semantic.coarse_unit_id_seq owned by semantic.coarse_unit.id;

create or replace function semantic.coarse_unit_validate_fine_ids()
returns trigger
language plpgsql
as $$
declare
  missing_id bigint;
begin
  select x
    into missing_id
    from unnest(new.fine_unit_ids) as x
   where not exists (
         select 1 from semantic.fine_unit f where f.id = x
   )
   limit 1;

  if missing_id is not null then
    raise exception 'fine_unit id % does not exist', missing_id;
  end if;

  new.updated_at := now();
  return new;
end;
$$;

drop trigger if exists trg_coarse_unit_validate on semantic.coarse_unit;

create trigger trg_coarse_unit_validate
before insert or update on semantic.coarse_unit
for each row execute function semantic.coarse_unit_validate_fine_ids();

create index if not exists ix_coarse_kind
on semantic.coarse_unit (kind);

create index if not exists ix_coarse_label
on semantic.coarse_unit (label);

create index if not exists coarse_unit_label_lower_idx
on semantic.coarse_unit (lower(label));

create index if not exists coarse_unit_label_lower_trgm_idx
on semantic.coarse_unit using gin (lower(label) gin_trgm_ops);

alter table semantic.coarse_unit enable row level security;

do $$
begin
  if exists (select 1 from pg_roles where rolname = 'authenticated')
     and exists (select 1 from pg_roles where rolname = 'anon')
     and not exists (
       select 1
       from pg_policies
       where schemaname = 'semantic'
         and tablename = 'coarse_unit'
         and policyname = 'coarse_unit_select_all'
     ) then
    execute 'create policy coarse_unit_select_all on semantic.coarse_unit for select to authenticated, anon using (true)';
  end if;
end;
$$;

create table if not exists semantic.unit_collections (
  collection_id uuid primary key default gen_random_uuid(),
  slug text not null unique
    constraint unit_collections_slug_canonical_check
    check (slug = lower(slug) and slug ~ '^[a-z0-9][a-z0-9-]{0,80}$'),
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
