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

	// Local development.
	_ = godotenv.Load()

	cfg := config.Load()

	ctx := context.Background()

	server, cleanup, err := internal.NewServer(
		ctx,
		cfg,
	)

	if err != nil {
		log.Fatalf(
			"failed to create server: %v",
			err,
		)
	}

	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	go func() {

		log.Printf(
			"FrameVerse server listening on :%s",
			cfg.Port,
		)

		if err := server.ListenAndServe(); err != nil {
			log.Printf(
				"server stopped: %v",
				err,
			)
		}
	}()

	<-stop

	log.Println("shutting down FrameVerse...")

	if err := server.Shutdown(
		context.Background(),
	); err != nil {

		log.Printf(
			"server shutdown error: %v",
			err,
		)
	}

	if err := cleanup(); err != nil {
		log.Printf(
			"cleanup error: %v",
			err,
		)
	}
}
