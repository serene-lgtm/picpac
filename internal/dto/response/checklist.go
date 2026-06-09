package response

// ChecklistLineItemResponse defines the API response for a checklist line item.
type ChecklistLineItemResponse struct {
	ID            string                `json:"id"`
	ReferenceID   string                `json:"reference_id"`
	ReferenceType string                `json:"reference_type"`
	Snapshot      *ItemSnapshotResponse `json:"snapshot"`
	Status        string                `json:"status"`
}

// ItemSnapshotResponse defines the API response for an item snapshot.
type ItemSnapshotResponse struct {
	Name string `json:"name"`
}

// ChecklistResponse defines the API response for a checklist.
type ChecklistResponse struct {
	ID          string                      `json:"id"`
	UserID      string                      `json:"user_id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	TargetDate  string                      `json:"target_date"`
	Items       []ChecklistLineItemResponse `json:"items"`
	Status      string                      `json:"status"`
}

// ListChecklistsResponse defines the API response for listing checklists.
type ListChecklistsResponse struct {
	Checklists []ChecklistResponse `json:"checklists"`
}

// DeleteChecklistResponse defines the API response for deleting a checklist.
type DeleteChecklistResponse struct {
	Deleted bool `json:"deleted"`
}
