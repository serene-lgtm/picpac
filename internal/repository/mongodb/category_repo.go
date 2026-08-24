package mongodb

import (
	"context"
	"time"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const categoryCollectionName = "categories"

type categoryDocument struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Key       string        `json:"key" bson:"key"`
	Name      string        `json:"name" bson:"nm"`
	CreatedAt time.Time     `json:"created_at" bson:"cat"`
	UpdatedAt time.Time     `json:"updated_at" bson:"uat"`
}

// CategoryRepository stores category domain models in MongoDB.
type CategoryRepository struct {
	collection *mongo.Collection
}

// NewCategoryRepository creates a MongoDB-backed category repository.
func NewCategoryRepository(db *mongo.Database) *CategoryRepository {
	return &CategoryRepository{collection: db.Collection(categoryCollectionName)}
}

// ListAll returns all system categories ordered by creation time.
func (r *CategoryRepository) ListAll(ctx context.Context) ([]domain.Category, error) {
	opts := options.Find().SetSort(bson.D{{Key: "cat", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []categoryDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	if docs == nil {
		return []domain.Category{}, nil
	}

	categories := make([]domain.Category, 0, len(docs))
	for _, doc := range docs {
		categories = append(categories, newDomainCategory(doc))
	}
	return categories, nil
}

// GetByID returns the category with the given ID.
func (r *CategoryRepository) GetByID(ctx context.Context, categoryID bson.ObjectID) (*domain.Category, error) {
	var doc categoryDocument
	if err := r.collection.FindOne(ctx, bson.M{"_id": categoryID}).Decode(&doc); err != nil {
		return nil, err
	}

	category := newDomainCategory(doc)
	return &category, nil
}

// GetByKey returns the category with the given key.
func (r *CategoryRepository) GetByKey(ctx context.Context, key string) (*domain.Category, error) {
	var doc categoryDocument
	if err := r.collection.FindOne(ctx, bson.M{"key": key}).Decode(&doc); err != nil {
		return nil, err
	}

	category := newDomainCategory(doc)
	return &category, nil
}

// UpsertByKey upserts a system category by key.
func (r *CategoryRepository) UpsertByKey(ctx context.Context, category *domain.Category) error {
	update := bson.M{
		"$set": bson.M{
			"nm":  category.Name,
			"uat": category.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"_id": category.ID,
			"key": category.Key,
			"cat": category.CreatedAt,
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"key": category.Key}, update, options.UpdateOne().SetUpsert(true))
	return err
}

func newDomainCategory(doc categoryDocument) domain.Category {
	return domain.Category{
		ID:        doc.ID,
		Key:       doc.Key,
		Name:      doc.Name,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}
