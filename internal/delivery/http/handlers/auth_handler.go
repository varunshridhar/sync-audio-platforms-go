package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync-audio-platforms-go/internal/infrastructure/spotify"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	spotifyClient *spotify.Client
	store         *sessions.CookieStore
}

func NewAuthHandler(spotifyClient *spotify.Client) *AuthHandler {
	// Generate a random key for cookie store
	key := make([]byte, 32)
	rand.Read(key)
	store := sessions.NewCookieStore(key)

	return &AuthHandler{
		spotifyClient: spotifyClient,
		store:         store,
	}
}

func (h *AuthHandler) Login(c echo.Context) error {
	// Generate random state
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.StdEncoding.EncodeToString(b)

	// Store state in session
	session, _ := h.store.Get(c.Request(), "spotify-auth")
	session.Values["state"] = state
	session.Save(c.Request(), c.Response().Writer)

	// Redirect to Spotify auth page
	authURL := h.spotifyClient.GetAuthURL(state)
	return c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *AuthHandler) Callback(c echo.Context) error {
	session, _ := h.store.Get(c.Request(), "spotify-auth")

	// Verify state
	state := c.QueryParam("state")
	sessionState, ok := session.Values["state"].(string)
	if !ok || state != sessionState {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "invalid state",
		})
	}

	// Exchange code for token
	code := c.QueryParam("code")
	token, err := h.spotifyClient.Exchange(code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to exchange code for token",
		})
	}

	// Store token in session
	session.Values["access_token"] = token.AccessToken
	session.Save(c.Request(), c.Response().Writer)

	// Redirect to frontend or return token
	return c.JSON(http.StatusOK, map[string]string{
		"access_token": token.AccessToken,
		"token_type":   token.TokenType,
	})
}
