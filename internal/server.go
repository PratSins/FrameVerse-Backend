package internal

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/PratSins/FrameVerse-Backend/config"
	"github.com/PratSins/FrameVerse-Backend/internal/toonify"
	appgcs "github.com/PratSins/FrameVerse-Backend/pkg/gcs"
	gemini "github.com/PratSins/FrameVerse-Backend/pkg/gemini"
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

	toonifyDAO := toonify.NewDAO(mongoClient)
	toonifyService := toonify.NewService(toonifyDAO, gcsClient, geminiClient)
	toonifyController := toonify.NewController(toonifyService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	toonifyController.MountRoutes(r)

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
