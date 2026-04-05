package enum

// UnitKind identifies the learning-unit kind stored in semantic.coarse_unit.
type UnitKind string

const (
	UnitKindWord    UnitKind = "word"
	UnitKindPhrase  UnitKind = "phrase"
	UnitKindGrammar UnitKind = "grammar"
)
