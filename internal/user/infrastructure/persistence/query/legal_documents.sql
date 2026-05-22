-- name: GetLegalDocument :one
select
  document_type,
  title,
  markdown,
  version,
  updated_at
from app_user.legal_documents
where document_type = sqlc.arg(document_type);
