package request

import "io"

// CreateItemInput defines the service input for creating an item.
type CreateItemInput struct {
	UserID      string
	CategoryID  string
	Name        string
	Description string
	File        io.ReadSeeker
	FileName    string
}

// BatchCreateItemInput defines one item in a batch create request.
type BatchCreateItemInput struct {
	CategoryID  string `json:"category_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// BatchCreateItemsInput defines the service input for batch creating items.
type BatchCreateItemsInput struct {
	UserID string                 `json:"-"`
	Items  []BatchCreateItemInput `json:"items"`
}

// ListItemsInput defines the service input for listing items.
type ListItemsInput struct {
	UserID        string
	Q             string
	HasQ          bool
	CategoryID    string
	HasCategoryID bool
}

// UpdateItemInput defines the service input for updating an item.
type UpdateItemInput struct {
	CategoryID    string
	HasCategoryID bool
	Name          string
	Description   string
	File          io.ReadSeeker
	FileName      string
}
