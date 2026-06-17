package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type AuthIdentity struct {
	ID         bson.ObjectID      `json:"id"`
	UserID     bson.ObjectID      `json:"user_id"`
	Provider   AuthProvider       `json:"provider"`
	Identifier string             `json:"identifier"`
	Status     AuthIdentityStatus `json:"status"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

type AuthProvider string

const (
	AuthProviderPhone  AuthProvider = "phone"
	AuthProviderWechat AuthProvider = "wechat"
)

type AuthIdentityStatus string

const (
	AuthIdentityStatusActive   AuthIdentityStatus = "active"
	AuthIdentityStatusDisabled AuthIdentityStatus = "disabled"
)

type PhoneVerificationCode struct {
	ID           bson.ObjectID            `json:"id"`
	Phone        string                   `json:"phone"`
	CodeHash     string                   `json:"code_hash"`
	Purpose      PhoneVerificationPurpose `json:"purpose"`
	ExpiresAt    time.Time                `json:"expires_at"`
	ConsumedAt   *time.Time               `json:"consumed_at"`
	AttemptCount int                      `json:"attempt_count"`
	CreatedAt    time.Time                `json:"created_at"`
}

type PhoneVerificationPurpose string

const (
	PhoneVerificationPurposeLogin PhoneVerificationPurpose = "login"
)

type RefreshToken struct {
	ID        bson.ObjectID `json:"id"`
	UserID    bson.ObjectID `json:"user_id"`
	TokenHash string        `json:"token_hash"`
	ExpiresAt time.Time     `json:"expires_at"`
	RevokedAt *time.Time    `json:"revoked_at"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
