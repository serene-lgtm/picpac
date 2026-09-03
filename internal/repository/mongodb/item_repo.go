package mongodb

import (
	"context"
	"regexp"
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const itemCollectionName = "items"

type itemDocument struct {
	ID                       bson.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID                   bson.ObjectID     `json:"user_id" bson:"uid"`
	CategoryID               bson.ObjectID     `json:"category_id" bson:"cid"`
	Name                     string            `json:"name" bson:"nm"`
	Description              string            `json:"description" bson:"desc"`
	SourceImageObjectKey     string            `json:"source_image_object_key" bson:"siok"`
	ImageThumbnailObjectKey  string            `json:"image_thumbnail_object_key" bson:"itok"`
	AIRenderedImageObjectKey string            `json:"ai_rendered_image_object_key" bson:"aiok"`
	Status                   domain.ItemStatus `json:"status" bson:"st"`
	CreatedAt                time.Time         `json:"created_at" bson:"cat"`
	UpdatedAt                time.Time         `json:"updated_at" bson:"uat"`
}

// ItemRepository stores item domain models in MongoDB.
type ItemRepository struct {
	collection *mongo.Collection
}

// NewItemRepository creates a MongoDB-backed item repository.
func NewItemRepository(db *mongo.Database) *ItemRepository {
	return &ItemRepository{
		collection: db.Collection(itemCollectionName),
	}
}

// Create inserts an item document into MongoDB.
func (r *ItemRepository) Create(ctx context.Context, item *domain.Item) error {
	_, err := r.collection.InsertOne(ctx, newItemDocument(item))
	return err
}

// CreateMany inserts item documents into MongoDB in one transaction.
func (r *ItemRepository) CreateMany(ctx context.Context, items []domain.Item) error {
	if len(items) == 0 {
		return nil
	}

	session, err := r.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessionCtx context.Context) (any, error) {
		docs := make([]any, 0, len(items))
		for index := range items {
			docs = append(docs, newItemDocument(&items[index]))
		}
		_, err := r.collection.InsertMany(sessionCtx, docs)
		return nil, err
	})
	return err
}

// ListAll returns all non-deleted items ordered by creation time descending.
func (r *ItemRepository) ListAll(ctx context.Context) ([]domain.Item, error) {
	docs, err := r.find(ctx, bson.M{"st": bson.M{"$ne": domain.ItemStatusDeleted}})
	if err != nil {
		return nil, err
	}

	return newDomainItems(docs), nil
}

// ListByUserID returns all items owned by the given user ordered by creation time descending.
func (r *ItemRepository) ListByUserID(ctx context.Context, userID bson.ObjectID) ([]domain.Item, error) {
	docs, err := r.find(ctx, bson.M{
		"uid": userID,
		"st":  bson.M{"$ne": domain.ItemStatusDeleted},
	})
	if err != nil {
		return nil, err
	}

	return newDomainItems(docs), nil
}

// ListByFilter returns non-deleted user items matching the given filters.
func (r *ItemRepository) ListByFilter(ctx context.Context, itemFilter repository.ItemFilter) ([]domain.Item, error) {
	filter := bson.M{
		"uid": itemFilter.UserID,
		"st":  bson.M{"$ne": domain.ItemStatusDeleted},
	}
	if itemFilter.HasCategoryID {
		filter["cid"] = itemFilter.CategoryID
	}
	if itemFilter.HasKeyword {
		filter["$or"] = itemKeywordConditions(itemFilter.Keyword)
	}

	docs, err := r.find(ctx, filter)
	if err != nil {
		return nil, err
	}

	return newDomainItems(docs), nil
}

// SearchByKeyword returns non-deleted items whose names or descriptions contain the keyword.
func (r *ItemRepository) SearchByKeyword(ctx context.Context, keyword string) ([]domain.Item, error) {
	docs, err := r.find(ctx, itemKeywordSearchFilter(keyword))
	if err != nil {
		return nil, err
	}

	return newDomainItems(docs), nil
}

// SearchByKeywordAndUserID returns non-deleted user items whose names or descriptions contain the keyword.
func (r *ItemRepository) SearchByKeywordAndUserID(ctx context.Context, userID bson.ObjectID, keyword string) ([]domain.Item, error) {
	filter := itemKeywordSearchFilter(keyword)
	filter["uid"] = userID

	docs, err := r.find(ctx, filter)
	if err != nil {
		return nil, err
	}

	return newDomainItems(docs), nil
}

// GetByID returns the item with the given ID.
func (r *ItemRepository) GetByID(ctx context.Context, itemID bson.ObjectID) (*domain.Item, error) {
	var doc itemDocument
	if err := r.collection.FindOne(ctx, bson.M{"_id": itemID}).Decode(&doc); err != nil {
		return nil, err
	}

	item := newDomainItem(doc)
	return &item, nil
}

// Update replaces the item document with the given ID.
func (r *ItemRepository) Update(ctx context.Context, item *domain.Item) error {
	doc := newItemDocument(item)
	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// DeleteByID logically deletes the item with the given ID.
func (r *ItemRepository) DeleteByID(ctx context.Context, itemID bson.ObjectID) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{
		"_id": itemID,
		"st":  bson.M{"$ne": domain.ItemStatusDeleted},
	}, bson.M{
		"$set": bson.M{
			"st":  domain.ItemStatusDeleted,
			"uat": time.Now().UTC(),
		},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func itemKeywordSearchFilter(keyword string) bson.M {
	return bson.M{
		"$or": itemKeywordConditions(keyword),
		"st":  bson.M{"$ne": domain.ItemStatusDeleted},
	}
}

func itemKeywordConditions(keyword string) bson.A {
	keywordPattern := bson.M{
		"$regex":   regexp.QuoteMeta(keyword),
		"$options": "i",
	}

	return bson.A{
		bson.M{"nm": keywordPattern},
		bson.M{"desc": keywordPattern},
	}
}

func (r *ItemRepository) find(ctx context.Context, filter bson.M) ([]itemDocument, error) {
	opts := options.Find().SetSort(bson.D{{Key: "cat", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []itemDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	if docs == nil {
		return []itemDocument{}, nil
	}

	return docs, nil
}

func newItemDocument(item *domain.Item) itemDocument {
	return itemDocument{
		ID:                       item.ID,
		UserID:                   item.UserID,
		CategoryID:               item.CategoryID,
		Name:                     item.Name,
		Description:              item.Description,
		SourceImageObjectKey:     item.SourceImageObjectKey,
		ImageThumbnailObjectKey:  item.ImageThumbnailObjectKey,
		AIRenderedImageObjectKey: item.AIRenderedImageObjectKey,
		Status:                   item.Status,
		CreatedAt:                item.CreatedAt,
		UpdatedAt:                item.UpdatedAt,
	}
}

func newDomainItem(doc itemDocument) domain.Item {
	return domain.Item{
		ID:                       doc.ID,
		UserID:                   doc.UserID,
		CategoryID:               doc.CategoryID,
		Name:                     doc.Name,
		Description:              doc.Description,
		SourceImageObjectKey:     doc.SourceImageObjectKey,
		ImageThumbnailObjectKey:  doc.ImageThumbnailObjectKey,
		AIRenderedImageObjectKey: doc.AIRenderedImageObjectKey,
		Status:                   doc.Status,
		CreatedAt:                doc.CreatedAt,
		UpdatedAt:                doc.UpdatedAt,
	}
}

func newDomainItems(docs []itemDocument) []domain.Item {
	items := make([]domain.Item, 0, len(docs))
	for _, doc := range docs {
		items = append(items, newDomainItem(doc))
	}

	return items
}
