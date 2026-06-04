package request

import "io"

// CreateItemInput defines the service input for creating an item.
type CreateItemInput struct {
	UserID      string
	Name        string
	Description string
	File        io.ReadSeeker
	FileName    string
}

// ListItemsInput defines the service input for listing items.
type ListItemsInput struct {
	UserID string
	Q      string
	HasQ   bool
}

// UpdateItemInput defines the service input for updating an item.
type UpdateItemInput struct {
	Name        string
	Description string
	File        io.ReadSeeker
	FileName    string
}
