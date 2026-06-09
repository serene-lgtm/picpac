package request

// ChecklistLineItemInput defines the service input for a checklist line item.
type ChecklistLineItemInput struct {
	ReferenceID   string             `json:"reference_id"`
	ReferenceType string             `json:"reference_type"`
	Snapshot      *ItemSnapshotInput `json:"snapshot"`
}

// ItemSnapshotInput defines the service input for an item snapshot.
type ItemSnapshotInput struct {
	Name string `json:"name"`
}

// CreateChecklistInput defines the service input for creating a checklist.
type CreateChecklistInput struct {
	UserID      string                   `json:"user_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	TargetDate  string                   `json:"target_date"`
	Items       []ChecklistLineItemInput `json:"items"`
}

// ListChecklistsInput defines the service input for listing checklists.
type ListChecklistsInput struct {
	UserID string
	Q      string
	HasQ   bool
}

// UpdateChecklistInput defines the service input for updating checklist metadata.
type UpdateChecklistInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TargetDate  string `json:"target_date"`
}

// AddChecklistLineItemsInput defines the service input for adding checklist line items.
type AddChecklistLineItemsInput struct {
	Items []ChecklistLineItemInput `json:"items"`
}

// RemoveChecklistLineItemsInput defines the service input for removing checklist line items.
type RemoveChecklistLineItemsInput struct {
	LineItemIDs []string `json:"line_item_ids"`
}
