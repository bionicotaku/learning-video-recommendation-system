package model

import "time"

const (
	LegalDocumentTypePrivacyPolicy = "privacy-policy"
	LegalDocumentTypeUserAgreement = "user-agreement"
)

type LegalDocument struct {
	Type      string
	Title     string
	Markdown  string
	Version   *string
	UpdatedAt *time.Time
}

func IsLegalDocumentType(value string) bool {
	return value == LegalDocumentTypePrivacyPolicy || value == LegalDocumentTypeUserAgreement
}
