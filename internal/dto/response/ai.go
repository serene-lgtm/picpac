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
