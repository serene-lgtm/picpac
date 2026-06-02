package request

// CreatePackInput defines the service input for creating a pack.
type CreatePackInput struct {
	Name        string   `json:"name"`
	UserID      string   `json:"user_id"`
	Description string   `json:"description"`
	Items       []string `json:"items"`
}

// UpdatePackInput defines the service input for updating a pack.
type UpdatePackInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Items       []string `json:"items"`
}
