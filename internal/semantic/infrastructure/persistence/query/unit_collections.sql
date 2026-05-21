-- name: ListActiveUnitCollections :many
select
  collection_id,
  slug,
  name,
  description,
  category,
  coarse_unit_count,
  word_unit_count
from semantic.unit_collections
where status = 'active'
order by category asc, name asc, slug asc;
