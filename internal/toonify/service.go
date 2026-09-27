package toonify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/PratSins/FrameVerse-Backend/pkg/gcs"
	gemini "github.com/PratSins/FrameVerse-Backend/pkg/gemini"
)

type Service struct {
	dao    *DAO
	gcs    *gcs.Client
	gemini *gemini.GeminiClient
}

func NewService(
	dao *DAO,
	gcsClient *gcs.Client,
	geminiClient *gemini.GeminiClient,
) *Service {

	return &Service{
		dao:    dao,
		gcs:    gcsClient,
		gemini: geminiClient,
	}
}

func (s *Service) CreateUpload(
	ctx context.Context,
	contentType string,
	style string,
	userID string,
) (*CreateUploadResponse, error) {

	jobID := uuid.New().String()

	objectName := fmt.Sprintf("uploads/%s/%s.mp4", jobID, jobID)

	job := &ToonifyJob{
		ID:          jobID,
		UserID:      userID,
		Status:      StatusPending,
		Style:       style,
		InputObject: objectName,
		CreatedAt:   time.Now(),
	}

	if err := s.dao.Create(ctx, job); err != nil {
		return nil, err
	}

	url, err := s.gcs.GenerateUploadURL(objectName, contentType)
	if err != nil {
		return nil, err
	}

	return &CreateUploadResponse{
		JobID:     jobID,
		UploadURL: url,
		Object:    objectName,
	}, nil
}

func (s *Service) Process(
	ctx context.Context,
	jobID string,
) error {

	job, err := s.dao.Get(ctx, jobID)
	if err != nil {
		return err
	}

	if err := s.dao.UpdateStatus(ctx, jobID, StatusProcessing, ""); err != nil {
		return err
	}

	inputGCSURI := fmt.Sprintf("gs://%s/%s", s.gcs.Bucket(), job.InputObject)
	outputGCSURI := fmt.Sprintf("gs://%s/processed/%s/", s.gcs.Bucket(), jobID)

	geminiOutput, err := s.gemini.Toonify(ctx, inputGCSURI, outputGCSURI, job.Style)
	if err != nil {
		_ = s.dao.UpdateStatus(ctx, jobID, StatusFailed, err.Error())
		return err
	}

	outputObject := strings.TrimPrefix(geminiOutput, fmt.Sprintf("gs://%s/", s.gcs.Bucket()))
	if outputObject == "" || outputObject == geminiOutput {
		outputObject = fmt.Sprintf("processed/%s/%s.mp4", jobID, jobID)
	}

	_, err = s.dao.collection.UpdateOne(
		ctx,
		map[string]interface{}{
			"_id": jobID,
		},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"status":        StatusCompleted,
				"output_object": outputObject,
				"completed_at":  time.Now(),
			},
		},
	)

	return err
}
