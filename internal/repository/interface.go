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
