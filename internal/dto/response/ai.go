package response

// RecommendedItemResponse defines one recommended item summary.
type RecommendedItemResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RecommendPackItemsResponse defines recommended existing item summaries for a pack.
type RecommendPackItemsResponse struct {
	RecommendedItems []RecommendedItemResponse `json:"recommended_items"`
}

// ItemDraftResponse defines one AI-extracted item draft.
type ItemDraftResponse struct {
	Name         string `json:"name"`
	CategoryID   string `json:"category_id"`
	CategoryKey  string `json:"category_key"`
	CategoryName string `json:"category_name"`
}

// GenerateItemDraftsResponse defines item drafts extracted from natural language.
type GenerateItemDraftsResponse struct {
	DraftItems []ItemDraftResponse `json:"draft_items"`
}
