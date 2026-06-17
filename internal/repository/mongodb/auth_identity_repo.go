package mongodb

import (
	"context"
	"time"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const authIdentityCollectionName = "auth_identities"

type authIdentityDocument struct {
	ID         bson.ObjectID             `json:"id" bson:"_id,omitempty"`
	UserID     bson.ObjectID             `json:"user_id" bson:"uid"`
	Provider   domain.AuthProvider       `json:"provider" bson:"prv"`
	Identifier string                    `json:"identifier" bson:"idf"`
	Status     domain.AuthIdentityStatus `json:"status" bson:"st"`
	CreatedAt  time.Time                 `json:"created_at" bson:"cat"`
	UpdatedAt  time.Time                 `json:"updated_at" bson:"uat"`
}

// AuthIdentityRepository stores auth identity domain models in MongoDB.
type AuthIdentityRepository struct {
	collection *mongo.Collection
}

// NewAuthIdentityRepository creates a MongoDB-backed auth identity repository.
func NewAuthIdentityRepository(db *mongo.Database) *AuthIdentityRepository {
	return &AuthIdentityRepository{collection: db.Collection(authIdentityCollectionName)}
}

// Create inserts an auth identity document into MongoDB.
func (r *AuthIdentityRepository) Create(ctx context.Context, identity *domain.AuthIdentity) error {
	_, err := r.collection.InsertOne(ctx, newAuthIdentityDocument(identity))
	return err
}

// GetByProviderAndIdentifier returns an auth identity by provider and identifier.
func (r *AuthIdentityRepository) GetByProviderAndIdentifier(ctx context.Context, provider domain.AuthProvider, identifier string) (*domain.AuthIdentity, error) {
	var doc authIdentityDocument
	if err := r.collection.FindOne(ctx, bson.M{
		"prv": provider,
		"idf": identifier,
		"st":  domain.AuthIdentityStatusActive,
	}).Decode(&doc); err != nil {
		return nil, err
	}

	identity := newDomainAuthIdentity(doc)
	return &identity, nil
}

func newAuthIdentityDocument(identity *domain.AuthIdentity) authIdentityDocument {
	return authIdentityDocument{
		ID:         identity.ID,
		UserID:     identity.UserID,
		Provider:   identity.Provider,
		Identifier: identity.Identifier,
		Status:     identity.Status,
		CreatedAt:  identity.CreatedAt,
		UpdatedAt:  identity.UpdatedAt,
	}
}

func newDomainAuthIdentity(doc authIdentityDocument) domain.AuthIdentity {
	return domain.AuthIdentity{
		ID:         doc.ID,
		UserID:     doc.UserID,
		Provider:   doc.Provider,
		Identifier: doc.Identifier,
		Status:     doc.Status,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
	}
}
