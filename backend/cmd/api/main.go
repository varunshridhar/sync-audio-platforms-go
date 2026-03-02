package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/varun/sync-audio-platforms-go/backend/internal/app"
	"github.com/varun/sync-audio-platforms-go/backend/internal/config"
)

// main is the backend entrypoint:
// 1) load validated configuration,
// 2) build the application dependencies,
// 3) start HTTP server with sane timeouts,
// 4) gracefully shutdown on SIGINT/SIGTERM.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("new app: %v", err)
	}
	defer application.Close()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           application.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Run the listener in a goroutine so the main goroutine can wait for OS signals.
	go func() {
		log.Printf("api listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Wait for termination signal and perform graceful shutdown with timeout.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}

