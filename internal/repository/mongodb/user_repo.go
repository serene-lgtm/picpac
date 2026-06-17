package mongodb

import (
	"context"
	"time"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const userCollectionName = "users"

type userDocument struct {
	ID          bson.ObjectID     `json:"id" bson:"_id,omitempty"`
	DisplayName string            `json:"display_name" bson:"dnm"`
	AvatarURL   string            `json:"avatar_url" bson:"avt"`
	Status      domain.UserStatus `json:"status" bson:"st"`
	CreatedAt   time.Time         `json:"created_at" bson:"cat"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"uat"`
}

// UserRepository stores user domain models in MongoDB.
type UserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository creates a MongoDB-backed user repository.
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{collection: db.Collection(userCollectionName)}
}

// Create inserts a user document into MongoDB.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.collection.InsertOne(ctx, newUserDocument(user))
	return err
}

// GetByID returns the non-deleted user with the given ID.
func (r *UserRepository) GetByID(ctx context.Context, userID bson.ObjectID) (*domain.User, error) {
	var doc userDocument
	if err := r.collection.FindOne(ctx, bson.M{
		"_id": userID,
		"st":  bson.M{"$ne": domain.UserStatusDeleted},
	}).Decode(&doc); err != nil {
		return nil, err
	}

	user := newDomainUser(doc)
	return &user, nil
}

func newUserDocument(user *domain.User) userDocument {
	return userDocument{
		ID:          user.ID,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func newDomainUser(doc userDocument) domain.User {
	return domain.User{
		ID:          doc.ID,
		DisplayName: doc.DisplayName,
		AvatarURL:   doc.AvatarURL,
		Status:      doc.Status,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}
