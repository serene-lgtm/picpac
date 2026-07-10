package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID        bson.ObjectID `json:"id"`
	Profile   UserProfile   `json:"profile"`
	Status    UserStatus    `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type UserProfile struct {
	Username        string     `json:"username"`
	Gender          UserGender `json:"gender"`
	Birthday        *time.Time `json:"birthday"`
	AvatarObjectKey string     `json:"avatar_object_key"`
}

type UserGender string

const (
	UserGenderMale    UserGender = "male"
	UserGenderFemale  UserGender = "female"
	UserGenderPrivate UserGender = "private"
)

type UserStatus string

const (
	UserStatusCreated  UserStatus = "created"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusDeleted  UserStatus = "deleted"
)
