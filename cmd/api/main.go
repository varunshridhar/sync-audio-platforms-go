package main

import (
	"sync-audio-platforms-go/internal/config"
	"sync-audio-platforms-go/internal/delivery/http/handlers"
	"sync-audio-platforms-go/internal/infrastructure/spotify"
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

	spotifyClient := spotify.NewSpotifyClient()
	spotifyUseCase := usecase.NewSpotifyUseCase(spotifyClient)
	spotifyHandler := handlers.NewSpotifyHandler(spotifyUseCase)
	authHandler := handlers.NewAuthHandler(spotifyClient)

	// Routes
	e.GET("/health", healthHandler.Check)
	e.GET("/auth/login", authHandler.Login)
	e.GET("/callback", authHandler.Callback)
	e.GET("/playlists", spotifyHandler.GetUserPlaylists)

	// Start server using the configured app port from the config module.
	e.Logger.Fatal(e.Start(":" + config.App.Port))
}
