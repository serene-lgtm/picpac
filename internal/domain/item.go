package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Item struct {
	ID                       bson.ObjectID `json:"id"`
	UserID                   bson.ObjectID `json:"user_id"`
	CategoryID               bson.ObjectID `json:"category_id"`
	Name                     string        `json:"name"`
	Description              string        `json:"description"`
	SourceImageObjectKey     string        `json:"source_image_object_key"`
	ImageThumbnailObjectKey  string        `json:"image_thumbnail_object_key"`
	AIRenderedImageObjectKey string        `json:"ai_rendered_image_object_key"`
	Status                   ItemStatus    `json:"status"`
	CreatedAt                time.Time     `json:"created_at"`
	UpdatedAt                time.Time     `json:"updated_at"`
}

type ItemStatus string

const (
	ItemStatusCreated ItemStatus = "created"
	ItemStatusDeleted ItemStatus = "deleted"
)
