package response

// ItemResponse defines the API response for an item.
type ItemResponse struct {
	ID                 string `json:"id"`
	UserID             string `json:"user_id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	SourceImageURL     string `json:"source_image_url"`
	ImageThumbnailURL  string `json:"image_thumbnail_url"`
	AIRenderedImageURL string `json:"ai_rendered_image_url"`
	Status             string `json:"status"`
}

// ListItemsResponse defines the API response for listing items.
type ListItemsResponse struct {
	Items []ItemResponse `json:"items"`
}

// DeleteItemResponse defines the API response for deleting an item.
type DeleteItemResponse struct {
	Deleted bool `json:"deleted"`
}
