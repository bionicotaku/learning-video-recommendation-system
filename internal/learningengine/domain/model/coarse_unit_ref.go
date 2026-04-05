package model

import "learning-video-recommendation-system/internal/learningengine/domain/enum"

// CoarseUnitRef is a lightweight reference to a semantic.coarse_unit record.
type CoarseUnitRef struct {
	CoarseUnitID int64
	Kind         enum.UnitKind
	Label        string
	Pos          string
	EnglishDef   string
	ChineseDef   string
}
