package internal

import (
	"context"
	"net/http"

	"github.com/PratSins/FrameVerse-Backend/config"
	"github.com/PratSins/FrameVerse-Backend/internal/toonify"
	appgcs "github.com/PratSins/FrameVerse-Backend/pkg/gcs"
	gemini "github.com/PratSins/FrameVerse-Backend/pkg/gemini"
	appmongo "github.com/PratSins/FrameVerse-Backend/pkg/mongo"
)

func NewServer(
	ctx context.Context,
	cfg *config.Config,
) (*http.Server, func() error, error) {

	gcsClient, err := appgcs.NewClient(
		ctx,
		cfg.GCSBucketName,
	)

	if err != nil {
		return nil, nil, err
	}

	mongoClient, err := appmongo.NewClient(
		ctx,
		cfg.MongoURI,
		cfg.MongoDatabase,
	)

	if err != nil {
		gcsClient.Close()
		return nil, nil, err
	}

	geminiClient, err := gemini.NewGeminiClient(
		ctx,
		cfg.GeminiAPIKey,
		cfg.GeminiModel,
	)

	if err != nil {
		gcsClient.Close()

		mongoClient.Close(ctx)

		return nil, nil, err
	}

	toonifyDAO := toonify.NewDAO(
		mongoClient,
	)

	toonifyService := toonify.NewService(
		toonifyDAO,
		gcsClient,
		geminiClient,
	)

	toonifyController := toonify.NewController(
		toonifyService,
	)

	mux := http.NewServeMux()

	mux.HandleFunc(
		"POST /api/v1/toonify/upload-url",
		toonifyController.CreateUploadURL,
	)

	mux.HandleFunc(
		"POST /api/v1/toonify/{jobID}/process",
		toonifyController.Process,
	)

	mux.HandleFunc(
		"GET /api/v1/toonify/{jobID}",
		toonifyController.GetJob,
	)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	cleanup := func() error {

		if err := gcsClient.Close(); err != nil {
			return err
		}

		return mongoClient.Close(context.Background())
	}

	return server, cleanup, nil
}
