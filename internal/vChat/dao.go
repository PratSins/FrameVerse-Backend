package vChat

import (
	"context"
	"errors"
	"fmt"
	"time"

	appmongo "github.com/PratSins/FrameVerse-Backend/pkg/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrRoomNotFound = errors.New("room not found")
)

type DAO struct {
	collection *mongo.Collection
}

func NewDAO(mongoClient *appmongo.Client) *DAO {
	return &DAO{
		collection: mongoClient.Database.Collection("rooms"),
	}
}

func (d *DAO) CreateRoom(ctx context.Context, room *Room) error {
	_, err := d.collection.InsertOne(ctx, room)
	if err != nil {
		return fmt.Errorf("failed to insert room: %w", err)
	}
	return nil
}

func (d *DAO) GetRoom(ctx context.Context, roomID string) (*Room, error) {
	var room Room
	err := d.collection.FindOne(ctx, bson.M{"_id": roomID}).Decode(&room)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("failed to get room: %w", err)
	}
	return &room, nil
}

func (d *DAO) CloseRoom(ctx context.Context, roomID string) error {
	now := time.Now()
	_, err := d.collection.UpdateOne(
		ctx,
		bson.M{"_id": roomID},
		bson.M{
			"$set": bson.M{
				"is_active": false,
				"ended_at":  now,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to close room: %w", err)
	}
	return nil
}
