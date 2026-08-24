package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Category struct {
	ID        bson.ObjectID `json:"id"`
	Key       string        `json:"key"`
	Name      string        `json:"name"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
