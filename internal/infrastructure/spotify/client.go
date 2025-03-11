package spotify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync-audio-platforms-go/internal/config"
	"sync-audio-platforms-go/internal/domain"

	"golang.org/x/oauth2"
)

type Client struct {
	httpClient   *http.Client
	baseURL      string
	oauth2Config *oauth2.Config
}

func NewSpotifyClient() *Client {
	oauth2Config := &oauth2.Config{
		ClientID:     config.Spotify.ClientID,
		ClientSecret: config.Spotify.ClientSecret,
		RedirectURL:  config.Spotify.RedirectURI,
		Scopes: []string{
			"playlist-read-private",
			"playlist-read-collaborative",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	}

	return &Client{
		httpClient:   &http.Client{},
		baseURL:      "https://api.spotify.com/v1",
		oauth2Config: oauth2Config,
	}
}

func (c *Client) GetAuthURL(state string) string {
	return c.oauth2Config.AuthCodeURL(state)
}

func (c *Client) Exchange(code string) (*oauth2.Token, error) {
	return c.oauth2Config.Exchange(oauth2.NoContext, code)
}

func (c *Client) GetUserPlaylists(accessToken string) ([]domain.SpotifyPlaylist, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/me/playlists", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var playlistResp domain.SpotifyPlaylistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&playlistResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return playlistResp.Items, nil
}

func (c *Client) GetPlaylistTracks(accessToken string, playlistID string) ([]domain.SpotifyTrack, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Items []struct {
			Track struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				URI   string `json:"uri"`
				Album struct {
					Name string `json:"name"`
				} `json:"album"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"track"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var tracks []domain.SpotifyTrack
	for _, item := range response.Items {
		track := domain.SpotifyTrack{
			ID:    item.Track.ID,
			Name:  item.Track.Name,
			URI:   item.Track.URI,
			Album: item.Track.Album.Name,
		}
		if len(item.Track.Artists) > 0 {
			track.Artist = item.Track.Artists[0].Name
		}
		tracks = append(tracks, track)
	}

	return tracks, nil
}
