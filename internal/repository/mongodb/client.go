package mongodb

import (
	"context"
	"fmt"
	"time"

	"pack_mate/internal/config"

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

	return &Connection{
		Client:   client,
		Database: client.Database(cfg.Database),
	}, nil
}

// Close disconnects the MongoDB client.
func (c *Connection) Close(ctx context.Context) error {
	if c == nil || c.Client == nil {
		return nil
	}
	return c.Client.Disconnect(ctx)
}
