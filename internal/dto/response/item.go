package response

// ItemResponse defines the API response for an item.
type ItemResponse struct {
	ID            string              `json:"id"`
	UserID        string              `json:"user_id"`
	CategoryID    string              `json:"category_id"`
	CategoryKey   string              `json:"category_key"`
	CategoryName  string              `json:"category_name"`
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	CoverImageURL string              `json:"cover_image_url"`
	Photos        []ItemPhotoResponse `json:"photos"`
	Status        string              `json:"status"`
}

// ItemPhotoResponse defines one item photo response.
type ItemPhotoResponse struct {
	ID             string `json:"id"`
	SourceImageURL string `json:"source_image_url"`
	ImageURL       string `json:"image_url"`
}

// ListItemsResponse defines the API response for listing items.
type ListItemsResponse struct {
	Items []ItemResponse `json:"items"`
}

// BatchCreateItemsResponse defines the API response for batch creating items.
type BatchCreateItemsResponse struct {
	Items []ItemResponse `json:"items"`
}

// DeleteItemResponse defines the API response for deleting an item.
type DeleteItemResponse struct {
	Deleted bool `json:"deleted"`
}
