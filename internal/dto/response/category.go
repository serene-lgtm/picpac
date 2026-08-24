package response

// CategoryResponse defines the API response for a category.
type CategoryResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ListCategoriesResponse defines the API response for listing categories.
type ListCategoriesResponse struct {
	Categories []CategoryResponse `json:"categories"`
}
