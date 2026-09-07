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
	ID        bson.ObjectID       `json:"id" bson:"_id,omitempty"`
	Profile   userProfileDocument `json:"profile" bson:"pf"`
	Status    domain.UserStatus   `json:"status" bson:"st"`
	CreatedAt time.Time           `json:"created_at" bson:"cat"`
	UpdatedAt time.Time           `json:"updated_at" bson:"uat"`
}

type userProfileDocument struct {
	Username               string            `json:"username" bson:"unm"`
	Gender                 domain.UserGender `json:"gender" bson:"gdr"`
	Birthday               *time.Time        `json:"birthday" bson:"bdy"`
	AvatarObjectKey        string            `json:"avatar_object_key" bson:"aok"`
	AvatarDisplayObjectKey string            `json:"avatar_display_object_key" bson:"adok"`
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

// Update replaces the non-deleted user document with the given ID.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	doc := newUserDocument(user)

	result, err := r.collection.ReplaceOne(ctx, bson.M{
		"_id": user.ID,
		"st":  bson.M{"$ne": domain.UserStatusDeleted},
	}, doc)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func newUserDocument(user *domain.User) userDocument {
	return userDocument{
		ID:        user.ID,
		Profile:   newUserProfileDocument(user.Profile),
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func newDomainUser(doc userDocument) domain.User {
	return domain.User{
		ID:        doc.ID,
		Profile:   newDomainUserProfile(doc.Profile),
		Status:    doc.Status,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}

func newUserProfileDocument(profile domain.UserProfile) userProfileDocument {
	return userProfileDocument{
		Username:               profile.Username,
		Gender:                 profile.Gender,
		Birthday:               profile.Birthday,
		AvatarObjectKey:        profile.AvatarObjectKey,
		AvatarDisplayObjectKey: profile.AvatarDisplayObjectKey,
	}
}

func newDomainUserProfile(doc userProfileDocument) domain.UserProfile {
	return domain.UserProfile{
		Username:               doc.Username,
		Gender:                 doc.Gender,
		Birthday:               doc.Birthday,
		AvatarObjectKey:        doc.AvatarObjectKey,
		AvatarDisplayObjectKey: doc.AvatarDisplayObjectKey,
	}
}
