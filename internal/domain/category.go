package domain

import "go.mongodb.org/mongo-driver/v2/bson"

type Category struct {
	ID         bson.ObjectID  `json:"id"`
	Name       string         `json:"name"`
	Level      int            `json:"level"`
	ParentID   bson.ObjectID  `json:"parent_id"`
	Attributes []AttributeDef `json:"attributes"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

type AttributeDef struct {
	Name    string   `json:"name"`    // e.g. "color", "size", "material"
	Options []string `json:"options"` // e.g. ["red", "blue", "green"] for color
}
