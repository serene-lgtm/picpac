package request

// CreatePackInput defines the service input for creating a pack.
type CreatePackInput struct {
	Name        string   `json:"name"`
	UserID      string   `json:"user_id"`
	Description string   `json:"description"`
	Items       []string `json:"items"`
}

// ListPacksInput defines the service input for listing packs.
type ListPacksInput struct {
	UserID string
	Q      string
	HasQ   bool
}

// UpdatePackProfileInput defines the service input for updating a pack profile.
type UpdatePackProfileInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AddPackItemsInput defines the service input for adding pack items.
type AddPackItemsInput struct {
	Items []string `json:"items"`
}

// RemovePackItemsInput defines the service input for removing pack items.
type RemovePackItemsInput struct {
	Items []string `json:"items"`
}
