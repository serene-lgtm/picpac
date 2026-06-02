package response

// PackResponse defines the API response for a pack.
type PackResponse struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Items       []string `json:"items"`
	Status      string   `json:"status"`
}

// ListPacksResponse defines the API response for listing packs.
type ListPacksResponse struct {
	Packs []PackResponse `json:"packs"`
}

// DeletePackResponse defines the API response for deleting a pack.
type DeletePackResponse struct {
	Deleted bool `json:"deleted"`
}
