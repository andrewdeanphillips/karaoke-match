package playlist

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/andrewdeanphillips/karaoke-match/backend/internal/spotify"
)

// ErrInvalidPlaylistURL indicates the caller supplied something other than a
// valid Spotify playlist URL — a client error, distinguishable via errors.Is
// from upstream failures so the HTTP handler can answer with 400 rather than
// 500.
var ErrInvalidPlaylistURL = errors.New("invalid Spotify playlist URL")

// parsePlaylistID extracts the playlist ID from a Spotify playlist URL such
// as https://open.spotify.com/playlist/{id}?si=... — the only piece of the
// URL that Spotify's API actually needs.
func parsePlaylistID(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPlaylistURL, err)
	}

	if parsed.Host != "open.spotify.com" {
		return "", fmt.Errorf("%w: unexpected host %q", ErrInvalidPlaylistURL, parsed.Host)
	}

	const pathPrefix = "/playlist/"
	if !strings.HasPrefix(parsed.Path, pathPrefix) {
		return "", fmt.Errorf("%w: expected a playlist link", ErrInvalidPlaylistURL)
	}

	id := strings.TrimPrefix(parsed.Path, pathPrefix)
	if id == "" || strings.Contains(id, "/") {
		return "", fmt.Errorf("%w: missing playlist ID", ErrInvalidPlaylistURL)
	}

	return id, nil
}

// uniqueArtists returns every artist credited on any of the given tracks,
// each appearing once, in the order they were first encountered.
func uniqueArtists(tracks []spotify.Track) []string {
	seen := make(map[string]struct{})
	var artists []string

	for _, track := range tracks {
		for _, artist := range track.Artists {
			if _, ok := seen[artist]; ok {
				continue
			}
			seen[artist] = struct{}{}
			artists = append(artists, artist)
		}
	}

	return artists
}

// Service orchestrates a playlist import: parsing the playlist URL, fetching
// its tracks from Spotify, and extracting the artists credited on them.
type Service struct {
	spotify *spotify.Client
}

func NewService(spotifyClient *spotify.Client) *Service {
	return &Service{spotify: spotifyClient}
}

// Import returns the unique artists credited across every track in the
// playlist identified by the given Spotify playlist URL. accessToken
// authenticates the request as the visitor whose session resolved it — see
// spotify.AccessTokenFromContext.
func (s *Service) Import(ctx context.Context, playlistURL, accessToken string) ([]string, error) {
	id, err := parsePlaylistID(playlistURL)
	if err != nil {
		return nil, err
	}

	tracks, err := s.spotify.GetPlaylistTracks(ctx, id, accessToken)
	if err != nil {
		return nil, fmt.Errorf("fetching playlist tracks: %w", err)
	}

	return uniqueArtists(tracks), nil
}
