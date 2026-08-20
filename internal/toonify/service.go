package toonify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
) (*CreateUploadResponse, error) {

	jobID := uuid.New().String()

	objectName := fmt.Sprintf(
		"uploads/%s/%s.mp4",
		jobID,
		jobID,
	)

	job := &ToonifyJob{
		ID:          jobID,
		Status:      StatusPending,
		Style:       style,
		InputObject: objectName,
		CreatedAt:   time.Now(),
	}

	if err := s.dao.Create(ctx, job); err != nil {
		return nil, err
	}

	url, err := s.gcs.GenerateUploadURL(
		objectName,
		contentType,
	)

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

	if err := s.dao.UpdateStatus(
		ctx,
		jobID,
		StatusProcessing,
		"",
	); err != nil {
		return err
	}

	inputPath := filepath.Join(
		os.TempDir(),
		jobID+"-input.mp4",
	)

	defer os.Remove(inputPath)

	if err := s.gcs.Download(
		ctx,
		job.InputObject,
		inputPath,
	); err != nil {

		_ = s.dao.UpdateStatus(
			ctx,
			jobID,
			StatusFailed,
			err.Error(),
		)

		return err
	}

	geminiOutput, err := s.gemini.Toonify(
		ctx,
		inputPath,
		job.Style,
	)

	if err != nil {
		_ = s.dao.UpdateStatus(
			ctx,
			jobID,
			StatusFailed,
			err.Error(),
		)

		return err
	}

	outputObject := fmt.Sprintf(
		"processed/%s/%s.mp4",
		jobID,
		jobID,
	)

	// If Gemini returned a URL, download it.
	outputPath := filepath.Join(
		os.TempDir(),
		jobID+"-output.mp4",
	)

	defer os.Remove(outputPath)

	if err := downloadFile(
		ctx,
		geminiOutput,
		outputPath,
	); err != nil {

		_ = s.dao.UpdateStatus(
			ctx,
			jobID,
			StatusFailed,
			err.Error(),
		)

		return err
	}

	if err := s.gcs.Upload(
		ctx,
		outputObject,
		outputPath,
		"video/mp4",
	); err != nil {

		_ = s.dao.UpdateStatus(
			ctx,
			jobID,
			StatusFailed,
			err.Error(),
		)

		return err
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

func downloadFile(
	ctx context.Context,
	url string,
	destination string,
) error {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)

	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"download failed: HTTP %d",
			resp.StatusCode,
		)
	}

	file, err := os.Create(destination)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = io.Copy(file, resp.Body)

	return err
}
