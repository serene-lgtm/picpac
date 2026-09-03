package request

// RecommendPackItemsInput defines the request for recommending existing items for a pack.
type RecommendPackItemsInput struct {
	UserID      string `json:"-"`
	PackName    string `json:"pack_name"`
	Description string `json:"description"`
}

// GenerateItemDraftsInput defines the request for extracting item drafts from text.
type GenerateItemDraftsInput struct {
	UserID string `json:"-"`
	Text   string `json:"text"`
}
