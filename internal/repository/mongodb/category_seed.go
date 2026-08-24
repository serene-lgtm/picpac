package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pack_mate/internal/config"
	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// SeedCategories upserts configured system categories into MongoDB.
func SeedCategories(ctx context.Context, db *mongo.Database, seeds []config.CategorySeedConfig) error {
	repo := NewCategoryRepository(db)
	now := time.Now().UTC()

	for _, seed := range seeds {
		category := &domain.Category{
			ID:        bson.NewObjectID(),
			Key:       strings.TrimSpace(seed.Key),
			Name:      strings.TrimSpace(seed.Name),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := repo.UpsertByKey(ctx, category); err != nil {
			return fmt.Errorf("seed category %s failed: %w", category.Key, err)
		}
	}

	return nil
}
