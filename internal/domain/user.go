package domain

import "go.mongodb.org/mongo-driver/v2/bson"

type User struct {
	ID          bson.ObjectID `json:"id"`
	Username    string        `json:"username"`
	PhoneNumber string        `json:"phone_number"`
	Password    string        `json:"password"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
}
