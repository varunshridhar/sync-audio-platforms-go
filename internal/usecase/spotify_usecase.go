package usecase

import "sync-audio-platforms-go/internal/domain"

type SpotifyUseCase struct {
	spotifyClient domain.SpotifyService
}

func NewSpotifyUseCase(spotifyClient domain.SpotifyService) *SpotifyUseCase {
	return &SpotifyUseCase{
		spotifyClient: spotifyClient,
	}
}

func (u *SpotifyUseCase) GetUserPlaylists(accessToken string) ([]domain.SpotifyPlaylist, error) {
	playlists, err := u.spotifyClient.GetUserPlaylists(accessToken)
	if err != nil {
		return nil, err
	}

	// We don't need to modify the playlists here since the tracks metadata
	// is already included in the response
	return playlists, nil
}

// Add a method to get tracks for a specific playlist
func (u *SpotifyUseCase) GetPlaylistTracks(accessToken string, playlistID string) ([]domain.SpotifyTrack, error) {
	return u.spotifyClient.GetPlaylistTracks(accessToken, playlistID)
}
