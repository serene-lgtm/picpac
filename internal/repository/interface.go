package repository

import (
	"context"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ItemRepository defines persistence behavior for items.
type ItemRepository interface {
	Create(ctx context.Context, item *domain.Item) error
	ListAll(ctx context.Context) ([]domain.Item, error)
	ListByUserID(ctx context.Context, userID bson.ObjectID) ([]domain.Item, error)
	GetByID(ctx context.Context, itemID bson.ObjectID) (*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) error
	DeleteByID(ctx context.Context, itemID bson.ObjectID) error
}
