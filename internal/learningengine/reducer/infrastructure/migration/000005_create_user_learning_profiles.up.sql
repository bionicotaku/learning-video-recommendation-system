create table if not exists learning.user_learning_profiles (
  user_id uuid primary key references auth.users(id) on delete cascade,
  active_collection_id uuid not null references semantic.unit_collections(collection_id),
  active_collection_slug text not null,
  active_collection_activated_at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
