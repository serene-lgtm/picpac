package mongodb

import (
	"context"
	"regexp"
	"time"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const itemCollectionName = "items"

type itemDocument struct {
	ID                 bson.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID             bson.ObjectID     `json:"user_id" bson:"uid"`
	Name               string            `json:"name" bson:"nm"`
	Description        string            `json:"description" bson:"desc"`
	SourceImageURL     string            `json:"source_image_url" bson:"src"`
	ImageThumbnailURL  string            `json:"image_thumbnail_url" bson:"thb"`
	AIRenderedImageURL string            `json:"ai_rendered_image_url" bson:"air"`
	Status             domain.ItemStatus `json:"status" bson:"st"`
	CreatedAt          time.Time         `json:"created_at" bson:"cat"`
	UpdatedAt          time.Time         `json:"updated_at" bson:"uat"`
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
	keywordPattern := bson.M{
		"$regex":   regexp.QuoteMeta(keyword),
		"$options": "i",
	}

	return bson.M{
		"$or": bson.A{
			bson.M{"nm": keywordPattern},
			bson.M{"desc": keywordPattern},
		},
		"st": bson.M{"$ne": domain.ItemStatusDeleted},
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
		ID:                 item.ID,
		UserID:             item.UserID,
		Name:               item.Name,
		Description:        item.Description,
		SourceImageURL:     item.SourceImageURL,
		ImageThumbnailURL:  item.ImageThumbnailURL,
		AIRenderedImageURL: item.AIRenderedImageURL,
		Status:             item.Status,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func newDomainItem(doc itemDocument) domain.Item {
	return domain.Item{
		ID:                 doc.ID,
		UserID:             doc.UserID,
		Name:               doc.Name,
		Description:        doc.Description,
		SourceImageURL:     doc.SourceImageURL,
		ImageThumbnailURL:  doc.ImageThumbnailURL,
		AIRenderedImageURL: doc.AIRenderedImageURL,
		Status:             doc.Status,
		CreatedAt:          doc.CreatedAt,
		UpdatedAt:          doc.UpdatedAt,
	}
}

func newDomainItems(docs []itemDocument) []domain.Item {
	items := make([]domain.Item, 0, len(docs))
	for _, doc := range docs {
		items = append(items, newDomainItem(doc))
	}

	return items
}
