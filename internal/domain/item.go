package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Item struct {
	ID                 bson.ObjectID `json:"id"`
	UserID             bson.ObjectID `json:"user_id"`
	Name               string        `json:"name"`
	Description        string        `json:"description"`
	SourceImageURL     string        `json:"source_image_url"`
	ImageThumbnailURL  string        `json:"image_thumbnail_url"`
	AIRenderedImageURL string        `json:"ai_rendered_image_url"`
	Status             ItemStatus    `json:"status"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type ItemStatus string

const (
	ItemStatusCreated ItemStatus = "created"
	ItemStatusDeleted ItemStatus = "deleted"
)
