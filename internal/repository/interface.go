package repository

import (
	"context"
	"time"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ItemRepository defines persistence behavior for items.
type ItemRepository interface {
	Create(ctx context.Context, item *domain.Item) error
	ListAll(ctx context.Context) ([]domain.Item, error)
	ListByUserID(ctx context.Context, userID bson.ObjectID) ([]domain.Item, error)
	SearchByKeyword(ctx context.Context, keyword string) ([]domain.Item, error)
	SearchByKeywordAndUserID(ctx context.Context, userID bson.ObjectID, keyword string) ([]domain.Item, error)
	GetByID(ctx context.Context, itemID bson.ObjectID) (*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) error
	DeleteByID(ctx context.Context, itemID bson.ObjectID) error
}

// PackRepository defines persistence behavior for packs.
type PackRepository interface {
	Create(ctx context.Context, pack *domain.Pack) error
	ListAll(ctx context.Context) ([]domain.Pack, error)
	ListByUserID(ctx context.Context, userID bson.ObjectID) ([]domain.Pack, error)
	SearchByKeyword(ctx context.Context, keyword string) ([]domain.Pack, error)
	SearchByKeywordAndUserID(ctx context.Context, userID bson.ObjectID, keyword string) ([]domain.Pack, error)
	GetByID(ctx context.Context, packID bson.ObjectID) (*domain.Pack, error)
	Update(ctx context.Context, pack *domain.Pack) error
	DeleteByID(ctx context.Context, packID bson.ObjectID) error
}

// ChecklistRepository defines persistence behavior for checklists.
type ChecklistRepository interface {
	Create(ctx context.Context, checklist *domain.Checklist) error
	ListAll(ctx context.Context) ([]domain.Checklist, error)
	ListByUserID(ctx context.Context, userID bson.ObjectID) ([]domain.Checklist, error)
	SearchByKeyword(ctx context.Context, keyword string) ([]domain.Checklist, error)
	SearchByKeywordAndUserID(ctx context.Context, userID bson.ObjectID, keyword string) ([]domain.Checklist, error)
	GetByID(ctx context.Context, checklistID bson.ObjectID) (*domain.Checklist, error)
	Update(ctx context.Context, checklist *domain.Checklist) error
	UpdateLineItemStatus(ctx context.Context, checklistID bson.ObjectID, lineItemID bson.ObjectID, status domain.LineItemStatus, updatedAt time.Time) error
	DeleteByID(ctx context.Context, checklistID bson.ObjectID) error
}

// UserRepository defines persistence behavior for users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, userID bson.ObjectID) (*domain.User, error)
}

// AuthIdentityRepository defines persistence behavior for auth identities.
type AuthIdentityRepository interface {
	Create(ctx context.Context, identity *domain.AuthIdentity) error
	GetByProviderAndIdentifier(ctx context.Context, provider domain.AuthProvider, identifier string) (*domain.AuthIdentity, error)
}

// PhoneVerificationCodeRepository defines persistence behavior for phone verification codes.
type PhoneVerificationCodeRepository interface {
	Create(ctx context.Context, code *domain.PhoneVerificationCode) error
	GetLatestActive(ctx context.Context, phone string, purpose domain.PhoneVerificationPurpose, now time.Time) (*domain.PhoneVerificationCode, error)
	MarkConsumed(ctx context.Context, codeID bson.ObjectID, consumedAt time.Time) error
	IncrementAttempt(ctx context.Context, codeID bson.ObjectID) error
	CountRecent(ctx context.Context, phone string, since time.Time) (int64, error)
}

// RefreshTokenRepository defines persistence behavior for refresh tokens.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string, revokedAt time.Time) error
}
