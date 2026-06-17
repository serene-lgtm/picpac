package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID          bson.ObjectID `json:"id"`
	DisplayName string        `json:"display_name"`
	AvatarURL   string        `json:"avatar_url"`
	Status      UserStatus    `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type UserStatus string

const (
	UserStatusCreated  UserStatus = "created"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusDeleted  UserStatus = "deleted"
)
