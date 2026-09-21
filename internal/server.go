package internal

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/PratSins/FrameVerse-Backend/config"
	"github.com/PratSins/FrameVerse-Backend/internal/toonify"
	"github.com/PratSins/FrameVerse-Backend/internal/vChat"
	"github.com/PratSins/FrameVerse-Backend/pkg/authclient"
	appgcs "github.com/PratSins/FrameVerse-Backend/pkg/gcs"
	gemini "github.com/PratSins/FrameVerse-Backend/pkg/gemini"
	appmiddleware "github.com/PratSins/FrameVerse-Backend/pkg/middleware"
	appmongo "github.com/PratSins/FrameVerse-Backend/pkg/mongo"
)

func NewServer(ctx context.Context, cfg *config.Config) (*http.Server, func() error, error) {

	gcsClient, err := appgcs.NewClient(ctx, cfg.GCSBucketName)
	if err != nil {
		return nil, nil, err
	}

	mongoClient, err := appmongo.NewClient(ctx, cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		gcsClient.Close()
		return nil, nil, err
	}

	geminiClient, err := gemini.NewGeminiClient(ctx, cfg.GCPProjectID, cfg.GCPLocation, cfg.GeminiModel)
	if err != nil {
		gcsClient.Close()
		mongoClient.Close(ctx)
		return nil, nil, err
	}

	authClient, err := authclient.NewClient(cfg)
	if err != nil {
		gcsClient.Close()
		mongoClient.Close(ctx)
		return nil, nil, err
	}

	// 1. Toonify Feature Setup
	toonifyDAO := toonify.NewDAO(mongoClient)
	toonifyService := toonify.NewService(toonifyDAO, gcsClient, geminiClient)
	toonifyController := toonify.NewController(toonifyService)

	// 2. vChat Feature Setup
	vChatDAO := vChat.NewDAO(mongoClient)
	vChatHub := vChat.NewHub()
	go vChatHub.Run()
	vChatService := vChat.NewService(vChatDAO, vChatHub)
	vChatController := vChat.NewController(vChatService, authClient)

	// 3. Router & Middlewares
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(appmiddleware.CORS())
	r.Use(appmiddleware.OptionalAuth(authClient))

	// 4. Mount Routes
	toonifyController.MountRoutes(r)
	vChatController.MountRoutes(r, appmiddleware.OptionalAuth(authClient))

	// 5. Health Check Endpoint
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"frameverse-backend"}`))
	})

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	cleanup := func() error {
		if err := gcsClient.Close(); err != nil {
			return err
		}
		return mongoClient.Close(context.Background())
	}

	return server, cleanup, nil
}
