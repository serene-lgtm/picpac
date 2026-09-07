package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Item struct {
	ID          bson.ObjectID `json:"id"`
	UserID      bson.ObjectID `json:"user_id"`
	CategoryID  bson.ObjectID `json:"category_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Photos      []ItemPhoto   `json:"photos"`
	Status      ItemStatus    `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type ItemPhoto struct {
	ID               bson.ObjectID `json:"id"`
	SourceObjectKey  string        `json:"source_object_key"`
	DisplayObjectKey string        `json:"display_object_key"`
	CreatedAt        time.Time     `json:"created_at"`
}

type ItemStatus string

const (
	ItemStatusCreated ItemStatus = "created"
	ItemStatusDeleted ItemStatus = "deleted"
)
