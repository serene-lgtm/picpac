package mongodb

import (
	"context"
	"fmt"
	"time"

	"pack_mate/internal/config"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connection wraps the MongoDB client and selected database handle.
type Connection struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// New creates a MongoDB client, verifies connectivity, and returns the selected database handle.
func New(cfg config.MongoConfig) (*Connection, error) {
	clientOptions := options.Client().ApplyURI(cfg.URI)
	if cfg.MaxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ConnectTimeoutSeconds)*time.Second)
	defer cancel()

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	db := client.Database(cfg.Database)
	if err := ensureIndexes(ctx, db); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ensure mongo indexes: %w", err)
	}

	return &Connection{
		Client:   client,
		Database: db,
	}, nil
}

// Close disconnects the MongoDB client.
func (c *Connection) Close(ctx context.Context) error {
	if c == nil || c.Client == nil {
		return nil
	}
	return c.Client.Disconnect(ctx)
}

func ensureIndexes(ctx context.Context, db *mongo.Database) error {
	if _, err := db.Collection(authIdentityCollectionName).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "prv", Value: 1},
			{Key: "idf", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	if _, err := db.Collection(refreshTokenCollectionName).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "th", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	if _, err := db.Collection(phoneVerificationCodeCollectionName).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "ph", Value: 1},
			{Key: "pur", Value: 1},
			{Key: "cat", Value: -1},
		},
	}); err != nil {
		return err
	}

	return nil
}
