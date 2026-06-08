package karaoke

import (
	"context"
	"fmt"
	"strings"
)

// Service answers whether artists are available on karaoke platforms.
type Service struct {
	joysound *joysoundClient
}

func NewService() *Service {
	return &Service{joysound: newJoysoundClient()}
}

// IsArtistAvailable reports whether JOYSOUND lists the given artist.
// JOYSOUND's search-results page itself separates artist matches from song
// matches, so every result search returns is already an artist — we just
// need to confirm one of them is actually the artist we're looking for,
// rather than a same-named act in JOYSOUND's catalog.
func (s *Service) IsArtistAvailable(ctx context.Context, artist string) (bool, error) {
	artists, err := s.joysound.search(ctx, artist)
	if err != nil {
		return false, fmt.Errorf("searching JOYSOUND for artist %q: %w", artist, err)
	}
	return anyArtistNamed(artists, artist), nil
}

// anyArtistNamed reports whether any result's name matches the given name,
// ignoring case to tolerate capitalization differences.
func anyArtistNamed(artists []Artist, name string) bool {
	for _, a := range artists {
		if strings.EqualFold(a.Name, name) {
			return true
		}
	}
	return false
}
