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

const packCollectionName = "packs"

type packDocument struct {
	ID          bson.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID      bson.ObjectID     `json:"user_id" bson:"uid"`
	Name        string            `json:"name" bson:"nm"`
	Description string            `json:"description" bson:"desc"`
	Items       []bson.ObjectID   `json:"items" bson:"itm"`
	Status      domain.PackStatus `json:"status" bson:"st"`
	CreatedAt   time.Time         `json:"created_at" bson:"cat"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"uat"`
}

// PackRepository stores pack domain models in MongoDB.
type PackRepository struct {
	collection *mongo.Collection
}

// NewPackRepository creates a MongoDB-backed pack repository.
func NewPackRepository(db *mongo.Database) *PackRepository {
	return &PackRepository{
		collection: db.Collection(packCollectionName),
	}
}

// Create inserts a pack document into MongoDB.
func (r *PackRepository) Create(ctx context.Context, pack *domain.Pack) error {
	_, err := r.collection.InsertOne(ctx, newPackDocument(pack))
	return err
}

// ListAll returns all non-deleted packs ordered by creation time descending.
func (r *PackRepository) ListAll(ctx context.Context) ([]domain.Pack, error) {
	docs, err := r.find(ctx, bson.M{"st": bson.M{"$ne": domain.PackStatusDeleted}})
	if err != nil {
		return nil, err
	}

	return newDomainPacks(docs), nil
}

// ListByUserID returns all non-deleted packs owned by the given user ordered by creation time descending.
func (r *PackRepository) ListByUserID(ctx context.Context, userID bson.ObjectID) ([]domain.Pack, error) {
	docs, err := r.find(ctx, bson.M{
		"uid": userID,
		"st":  bson.M{"$ne": domain.PackStatusDeleted},
	})
	if err != nil {
		return nil, err
	}

	return newDomainPacks(docs), nil
}

// SearchByKeyword returns non-deleted packs whose names or descriptions contain the keyword.
func (r *PackRepository) SearchByKeyword(ctx context.Context, keyword string) ([]domain.Pack, error) {
	docs, err := r.find(ctx, packKeywordSearchFilter(keyword))
	if err != nil {
		return nil, err
	}

	return newDomainPacks(docs), nil
}

// SearchByKeywordAndUserID returns non-deleted user packs whose names or descriptions contain the keyword.
func (r *PackRepository) SearchByKeywordAndUserID(ctx context.Context, userID bson.ObjectID, keyword string) ([]domain.Pack, error) {
	filter := packKeywordSearchFilter(keyword)
	filter["uid"] = userID

	docs, err := r.find(ctx, filter)
	if err != nil {
		return nil, err
	}

	return newDomainPacks(docs), nil
}

// GetByID returns the non-deleted pack with the given ID.
func (r *PackRepository) GetByID(ctx context.Context, packID bson.ObjectID) (*domain.Pack, error) {
	var doc packDocument
	if err := r.collection.FindOne(ctx, bson.M{
		"_id": packID,
		"st":  bson.M{"$ne": domain.PackStatusDeleted},
	}).Decode(&doc); err != nil {
		return nil, err
	}

	pack := newDomainPack(doc)
	return &pack, nil
}

// Update replaces the non-deleted pack document with the given ID.
func (r *PackRepository) Update(ctx context.Context, pack *domain.Pack) error {
	doc := newPackDocument(pack)
	result, err := r.collection.ReplaceOne(ctx, bson.M{
		"_id": doc.ID,
		"st":  bson.M{"$ne": domain.PackStatusDeleted},
	}, doc)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// DeleteByID logically deletes the pack with the given ID.
func (r *PackRepository) DeleteByID(ctx context.Context, packID bson.ObjectID) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{
		"_id": packID,
		"st":  bson.M{"$ne": domain.PackStatusDeleted},
	}, bson.M{
		"$set": bson.M{
			"st":  domain.PackStatusDeleted,
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

func packKeywordSearchFilter(keyword string) bson.M {
	keywordPattern := bson.M{
		"$regex":   regexp.QuoteMeta(keyword),
		"$options": "i",
	}

	return bson.M{
		"$or": bson.A{
			bson.M{"nm": keywordPattern},
			bson.M{"desc": keywordPattern},
		},
		"st": bson.M{"$ne": domain.PackStatusDeleted},
	}
}

func (r *PackRepository) find(ctx context.Context, filter bson.M) ([]packDocument, error) {
	opts := options.Find().SetSort(bson.D{{Key: "cat", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []packDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	if docs == nil {
		return []packDocument{}, nil
	}

	return docs, nil
}

func newPackDocument(pack *domain.Pack) packDocument {
	return packDocument{
		ID:          pack.ID,
		UserID:      pack.UserID,
		Name:        pack.Name,
		Description: pack.Description,
		Items:       pack.Items,
		Status:      pack.Status,
		CreatedAt:   pack.CreatedAt,
		UpdatedAt:   pack.UpdatedAt,
	}
}

func newDomainPack(doc packDocument) domain.Pack {
	return domain.Pack{
		ID:          doc.ID,
		UserID:      doc.UserID,
		Name:        doc.Name,
		Description: doc.Description,
		Items:       doc.Items,
		Status:      doc.Status,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}

func newDomainPacks(docs []packDocument) []domain.Pack {
	packs := make([]domain.Pack, 0, len(docs))
	for _, doc := range docs {
		packs = append(packs, newDomainPack(doc))
	}

	return packs
}
