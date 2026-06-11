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

// trackKey identifies a track by its title and primary (first-credited)
// artist — the same pair karaoke.Track matches against, so deduping on it
// here avoids redundant JOYSOUND lookups for repeated tracks.
type trackKey struct {
	artist string
	title  string
}

// uniqueTracks returns every track from the given list, each appearing once,
// in the order first encountered. Tracks are deduplicated by their primary
// artist and title; tracks with no artists contribute nothing, since there's
// no artist to key or match against.
func uniqueTracks(tracks []spotify.Track) []spotify.Track {
	seen := make(map[trackKey]struct{})
	var unique []spotify.Track

	for _, track := range tracks {
		if len(track.Artists) == 0 {
			continue
		}

		key := trackKey{artist: track.Artists[0], title: track.Name}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, track)
	}

	return unique
}

// Service orchestrates a playlist import: parsing the playlist URL, fetching
// its tracks from Spotify, and extracting the artists credited on them.
type Service struct {
	spotify *spotify.Client
}

func NewService(spotifyClient *spotify.Client) *Service {
	return &Service{spotify: spotifyClient}
}

// Import returns the unique tracks in the playlist identified by the given
// Spotify playlist URL. accessToken authenticates the request as the visitor
// whose session resolved it — see spotify.AccessTokenFromContext.
func (s *Service) Import(ctx context.Context, playlistURL, accessToken string) ([]spotify.Track, error) {
	id, err := parsePlaylistID(playlistURL)
	if err != nil {
		return nil, err
	}

	tracks, err := s.spotify.GetPlaylistTracks(ctx, id, accessToken)
	if err != nil {
		return nil, fmt.Errorf("fetching playlist tracks: %w", err)
	}

	return uniqueTracks(tracks), nil
}
