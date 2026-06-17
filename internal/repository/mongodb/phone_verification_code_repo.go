package mongodb

import (
	"context"
	"time"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const phoneVerificationCodeCollectionName = "phone_verification_codes"

type phoneVerificationCodeDocument struct {
	ID           bson.ObjectID                   `json:"id" bson:"_id,omitempty"`
	Phone        string                          `json:"phone" bson:"ph"`
	CodeHash     string                          `json:"code_hash" bson:"ch"`
	Purpose      domain.PhoneVerificationPurpose `json:"purpose" bson:"pur"`
	ExpiresAt    time.Time                       `json:"expires_at" bson:"exp"`
	ConsumedAt   *time.Time                      `json:"consumed_at" bson:"cns,omitempty"`
	AttemptCount int                             `json:"attempt_count" bson:"att"`
	CreatedAt    time.Time                       `json:"created_at" bson:"cat"`
}

// PhoneVerificationCodeRepository stores phone verification codes in MongoDB.
type PhoneVerificationCodeRepository struct {
	collection *mongo.Collection
}

// NewPhoneVerificationCodeRepository creates a MongoDB-backed phone code repository.
func NewPhoneVerificationCodeRepository(db *mongo.Database) *PhoneVerificationCodeRepository {
	return &PhoneVerificationCodeRepository{collection: db.Collection(phoneVerificationCodeCollectionName)}
}

// Create inserts a phone verification code document into MongoDB.
func (r *PhoneVerificationCodeRepository) Create(ctx context.Context, code *domain.PhoneVerificationCode) error {
	_, err := r.collection.InsertOne(ctx, newPhoneVerificationCodeDocument(code))
	return err
}

// GetLatestActive returns the latest unconsumed, unexpired verification code.
func (r *PhoneVerificationCodeRepository) GetLatestActive(ctx context.Context, phone string, purpose domain.PhoneVerificationPurpose, now time.Time) (*domain.PhoneVerificationCode, error) {
	var doc phoneVerificationCodeDocument
	opts := options.FindOne().SetSort(bson.D{{Key: "cat", Value: -1}})
	if err := r.collection.FindOne(ctx, bson.M{
		"ph":  phone,
		"pur": purpose,
		"exp": bson.M{"$gt": now},
		"cns": bson.M{"$exists": false},
	}, opts).Decode(&doc); err != nil {
		return nil, err
	}

	code := newDomainPhoneVerificationCode(doc)
	return &code, nil
}

// MarkConsumed marks a phone verification code as consumed.
func (r *PhoneVerificationCodeRepository) MarkConsumed(ctx context.Context, codeID bson.ObjectID, consumedAt time.Time) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": codeID}, bson.M{
		"$set": bson.M{"cns": consumedAt},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// IncrementAttempt increments the verification attempt count.
func (r *PhoneVerificationCodeRepository) IncrementAttempt(ctx context.Context, codeID bson.ObjectID) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": codeID}, bson.M{
		"$inc": bson.M{"att": 1},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// CountRecent counts verification codes created since the given time.
func (r *PhoneVerificationCodeRepository) CountRecent(ctx context.Context, phone string, since time.Time) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"ph":  phone,
		"cat": bson.M{"$gte": since},
	})
}

func newPhoneVerificationCodeDocument(code *domain.PhoneVerificationCode) phoneVerificationCodeDocument {
	return phoneVerificationCodeDocument{
		ID:           code.ID,
		Phone:        code.Phone,
		CodeHash:     code.CodeHash,
		Purpose:      code.Purpose,
		ExpiresAt:    code.ExpiresAt,
		ConsumedAt:   code.ConsumedAt,
		AttemptCount: code.AttemptCount,
		CreatedAt:    code.CreatedAt,
	}
}

func newDomainPhoneVerificationCode(doc phoneVerificationCodeDocument) domain.PhoneVerificationCode {
	return domain.PhoneVerificationCode{
		ID:           doc.ID,
		Phone:        doc.Phone,
		CodeHash:     doc.CodeHash,
		Purpose:      doc.Purpose,
		ExpiresAt:    doc.ExpiresAt,
		ConsumedAt:   doc.ConsumedAt,
		AttemptCount: doc.AttemptCount,
		CreatedAt:    doc.CreatedAt,
	}
}
