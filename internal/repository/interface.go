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
