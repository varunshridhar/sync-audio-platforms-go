package domain

type SpotifyPlaylist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
	Images      []struct {
		URL string `json:"url"`
	} `json:"images"`
	Tracks struct {
		Href  string `json:"href"`
		Total int    `json:"total"`
	} `json:"tracks"`
}

type SpotifyTrack struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	URI    string `json:"uri"`
}

type SpotifyPlaylistsResponse struct {
	Items []SpotifyPlaylist `json:"items"`
	Total int               `json:"total"`
}

type SpotifyService interface {
	GetUserPlaylists(accessToken string) ([]SpotifyPlaylist, error)
	GetPlaylistTracks(accessToken string, playlistID string) ([]SpotifyTrack, error)
}
