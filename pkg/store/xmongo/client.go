package xmongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Client embeds the official MongoDB client.
type Client struct {
	*mongo.Client
	config *Config
}

func newClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("xmongo: nil config")
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("xmongo: empty dsn for %q", cfg.Name)
	}

	clientOptions := options.Client().ApplyURI(cfg.DSN)
	if cfg.MinPoolSize > 0 {
		clientOptions.SetMinPoolSize(cfg.MinPoolSize)
	}
	if cfg.MaxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
	}
	if cfg.ConnectTimeout > 0 {
		clientOptions.SetConnectTimeout(cfg.ConnectTimeout)
	}
	if cfg.ServerSelectionTimeout > 0 {
		clientOptions.SetServerSelectionTimeout(cfg.ServerSelectionTimeout)
	}

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("xmongo: connect: %w", err)
	}

	timeout := cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("xmongo: ping: %w", err)
	}
	return &Client{Client: client, config: cfg}, nil
}

// Config returns the client config.
func (c *Client) Config() *Config { return c.config }

// Collection returns a collection handle.
func (c *Client) Collection(database, collection string) *mongo.Collection {
	return c.Client.Database(database).Collection(collection)
}

// Ping verifies connectivity.
func (c *Client) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx, readpref.Primary())
}

// Close disconnects the client.
func (c *Client) Close(ctx context.Context) error {
	return c.Client.Disconnect(ctx)
}
