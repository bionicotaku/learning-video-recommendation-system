package dto

type GetLegalDocumentRequest struct {
	Type string
}

type GetLegalDocumentResponse struct {
	Type      string  `json:"type"`
	Title     string  `json:"title"`
	Markdown  string  `json:"markdown"`
	UpdatedAt *string `json:"updated_at"`
	Version   *string `json:"version"`
}
