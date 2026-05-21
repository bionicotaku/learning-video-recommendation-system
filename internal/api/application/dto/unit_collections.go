package dto

type ListUnitCollectionsRequest struct {
	UserID string
}

type UnitCollectionItem struct {
	CollectionID    string  `json:"collection_id"`
	Slug            string  `json:"slug"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	Category        string  `json:"category"`
	CoarseUnitCount int32   `json:"coarse_unit_count"`
	WordUnitCount   int32   `json:"word_unit_count"`
}

type UnitCollectionsResponse struct {
	Items            []UnitCollectionItem `json:"items"`
	ActiveCollection *string              `json:"active_collection"`
}
