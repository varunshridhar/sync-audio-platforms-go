package main

import (
	"sync-audio-platforms-go/internal/config"
	"sync-audio-platforms-go/internal/delivery/http/handlers"
	"sync-audio-platforms-go/internal/usecase"

	"github.com/labstack/echo/v4"
)

const version = "1.0.0"

func main() {
	// Load the configuration from environment variables.
	config.LoadConfig()

	// Initialize Echo
	e := echo.New()

	// Initialize dependencies
	healthUseCase := usecase.NewHealthUseCase(version)
	healthHandler := handlers.NewHealthHandler(healthUseCase)

	// Routes
	e.GET("/health", healthHandler.Check)

	// Start server using the configured app port from the config module.
	e.Logger.Fatal(e.Start(":" + config.App.Port))
}
