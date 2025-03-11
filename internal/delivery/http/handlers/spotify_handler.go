package handlers

import (
	"net/http"
	"sync-audio-platforms-go/internal/usecase"

	"github.com/labstack/echo/v4"
)

type SpotifyHandler struct {
	spotifyUseCase *usecase.SpotifyUseCase
}

func NewSpotifyHandler(spotifyUseCase *usecase.SpotifyUseCase) *SpotifyHandler {
	return &SpotifyHandler{
		spotifyUseCase: spotifyUseCase,
	}
}

func (h *SpotifyHandler) GetUserPlaylists(c echo.Context) error {
	// Get access token from Authorization header
	accessToken := c.Request().Header.Get("Authorization")
	if accessToken == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "missing access token",
		})
	}

	// Remove "Bearer " prefix if present
	if len(accessToken) > 7 && accessToken[:7] == "Bearer " {
		accessToken = accessToken[7:]
	}

	playlists, err := h.spotifyUseCase.GetUserPlaylists(accessToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, playlists)
}
