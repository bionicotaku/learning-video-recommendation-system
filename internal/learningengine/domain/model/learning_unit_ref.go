package model

import "learning-video-recommendation-system/internal/learningengine/domain/enum"

// LearningUnitRef is a lightweight reference to a learning unit.
type LearningUnitRef struct {
	CoarseUnitID int64
	Kind         enum.UnitKind
	Label        string
	Pos          string
	EnglishDef   string
	ChineseDef   string
}
