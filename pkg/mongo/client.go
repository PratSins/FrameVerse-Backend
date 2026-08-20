package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Client struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewClient(
	ctx context.Context,
	uri string,
	databaseName string,
) (*Client, error) {

	client, err := mongo.Connect(
		options.Client().ApplyURI(uri),
	)

	if err != nil {
		return nil, fmt.Errorf("connect to mongo: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return &Client{
		Client:   client,
		Database: client.Database(databaseName),
	}, nil
}

func (c *Client) Close(ctx context.Context) error {
	return c.Client.Disconnect(ctx)
}
