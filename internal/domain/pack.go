package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Pack struct {
	ID          bson.ObjectID   `json:"id"`
	UserID      bson.ObjectID   `json:"user_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Items       []bson.ObjectID `json:"items"`
	Status      PackStatus      `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type PackStatus string

const (
	PackStatusCreated PackStatus = "created"
	PackStatusDeleted PackStatus = "deleted"
)
