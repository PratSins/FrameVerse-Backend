package toonify

import (
	"context"
	"errors"
	"time"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	appmongo "github.com/PratSins/FrameVerse-Backend/pkg/mongo"
)

type DAO struct {
	collection *mongodriver.Collection
}

func NewDAO(client *appmongo.Client) *DAO {
	return &DAO{
		collection: client.Database.Collection("toonify_jobs"),
	}
}

func (d *DAO) Create(
	ctx context.Context,
	job *ToonifyJob,
) error {

	_, err := d.collection.InsertOne(ctx, job)

	return err
}

func (d *DAO) Get(
	ctx context.Context,
	jobID string,
) (*ToonifyJob, error) {

	var job ToonifyJob

	err := d.collection.FindOne(
		ctx,
		map[string]interface{}{
			"_id": jobID,
		},
	).Decode(&job)

	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return nil, errors.New("job not found")
		}

		return nil, err
	}

	return &job, nil
}

func (d *DAO) UpdateStatus(
	ctx context.Context,
	jobID string,
	status JobStatus,
	errMessage string,
) error {

	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"status": status,
			"error":  errMessage,
		},
	}

	if status == StatusCompleted {
		update["$set"].(map[string]interface{})["completed_at"] = time.Now()
	}

	_, err := d.collection.UpdateOne(
		ctx,
		map[string]interface{}{
			"_id": jobID,
		},
		update,
		options.UpdateOne(),
	)

	return err
}
