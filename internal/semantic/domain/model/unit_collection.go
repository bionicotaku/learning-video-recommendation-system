package model

type UnitCollection struct {
	CollectionID    string
	Slug            string
	Name            string
	Description     *string
	Category        string
	CoarseUnitCount int32
	WordUnitCount   int32
}
