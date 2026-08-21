package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/PratSins/FrameVerse-Backend/config"
	"github.com/PratSins/FrameVerse-Backend/internal"
)

func main() {
	// This line is for Local development. It reads from local .env file automatically
	// No need to comment it for production, as the .env file is going to be missing in the production server
	// The _ is used to ignore the error from godotenv.Load(), which is expected in production
	// The returned error is simple error value like "Production: Ignores the missing file and reads from real OS environment variables."
	// Which is not a critical error, Therefore it will not cause any crash
	// In Production, we will use environment variables from the cloud platform
	_ = godotenv.Load()

	cfg := config.Load()
	ctx := context.Background()
	server, cleanup, err := internal.NewServer(ctx, cfg) // Server Initiated Here
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	stop := make(chan os.Signal, 1) // Signal channel
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.Printf("FrameVerse server listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil { // Server starts here
			log.Printf("server stopped: %v", err)
		}
	}()
	<-stop

	log.Println("shutting down FrameVerse...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	if err := cleanup(); err != nil {
		log.Printf("cleanup error: %v", err)
	}
}
