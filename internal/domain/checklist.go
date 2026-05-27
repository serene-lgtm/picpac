package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Checklist struct {
	ID          bson.ObjectID `json:"id"`
	UserID      bson.ObjectID `json:"user_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	TargetDate  time.Time     `json:"target_date"`
	Items       []LineItem
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type LineItem struct {
	ID            bson.ObjectID `json:"id"`
	ReferenceID   bson.ObjectID
	ReferenceType LineItemType
	Status        LineItemStatus
}

type ItemSnapshot struct {
}

type LineItemType string

const (
	LineItemTypeItem     LineItemType = "item"
	LineItemTypeSnapshot LineItemType = "snapshot"
)

type LineItemStatus string

const (
	LineItemStatusCreated   LineItemStatus = "created"
	LineItemStatusChecked   LineItemStatus = "checked"
	LineItemStatusUnchecked LineItemStatus = "unchecked"
)
