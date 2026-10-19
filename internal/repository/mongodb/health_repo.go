package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// HealthRepository checks MongoDB availability.
type HealthRepository struct {
	client *mongo.Client
}

// NewHealthRepository creates a MongoDB health repository.
func NewHealthRepository(db *mongo.Database) *HealthRepository {
	return &HealthRepository{client: db.Client()}
}

// Ping checks whether MongoDB is reachable.
func (r *HealthRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx, nil)
}
