package mongodb

import (
	"context"
	"time"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const refreshTokenCollectionName = "refresh_tokens"

type refreshTokenDocument struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    bson.ObjectID `json:"user_id" bson:"uid"`
	TokenHash string        `json:"token_hash" bson:"th"`
	ExpiresAt time.Time     `json:"expires_at" bson:"exp"`
	RevokedAt *time.Time    `json:"revoked_at" bson:"rev,omitempty"`
	CreatedAt time.Time     `json:"created_at" bson:"cat"`
	UpdatedAt time.Time     `json:"updated_at" bson:"uat"`
}

// RefreshTokenRepository stores refresh tokens in MongoDB.
type RefreshTokenRepository struct {
	collection *mongo.Collection
}

// NewRefreshTokenRepository creates a MongoDB-backed refresh token repository.
func NewRefreshTokenRepository(db *mongo.Database) *RefreshTokenRepository {
	return &RefreshTokenRepository{collection: db.Collection(refreshTokenCollectionName)}
}

// Create inserts a refresh token document into MongoDB.
func (r *RefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	_, err := r.collection.InsertOne(ctx, newRefreshTokenDocument(token))
	return err
}

// GetByTokenHash returns a refresh token by token hash.
func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	var doc refreshTokenDocument
	if err := r.collection.FindOne(ctx, bson.M{"th": tokenHash}).Decode(&doc); err != nil {
		return nil, err
	}

	token := newDomainRefreshToken(doc)
	return &token, nil
}

// Revoke revokes a refresh token.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{"th": tokenHash}, bson.M{
		"$set": bson.M{
			"rev": revokedAt,
			"uat": revokedAt,
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

func newRefreshTokenDocument(token *domain.RefreshToken) refreshTokenDocument {
	return refreshTokenDocument{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		RevokedAt: token.RevokedAt,
		CreatedAt: token.CreatedAt,
		UpdatedAt: token.UpdatedAt,
	}
}

func newDomainRefreshToken(doc refreshTokenDocument) domain.RefreshToken {
	return domain.RefreshToken{
		ID:        doc.ID,
		UserID:    doc.UserID,
		TokenHash: doc.TokenHash,
		ExpiresAt: doc.ExpiresAt,
		RevokedAt: doc.RevokedAt,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}
