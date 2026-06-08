package karaoke

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// catalog is the subset of joysoundClient that Service depends on — narrow
// enough that tests can substitute a fake and exercise matching/aggregation
// logic without making live requests to JOYSOUND.
type catalog interface {
	search(ctx context.Context, keyword string) ([]Artist, error)
}

// pauseBetweenSearches keeps a multi-artist check considerate of JOYSOUND's
// servers: this is a sequence of individual page requests, not a bulk API,
// so we space them out rather than firing them as fast as possible.
const pauseBetweenSearches = 200 * time.Millisecond

// maxArtistsPerCheck bounds how many artists a single CheckAvailability call
// will look up — large playlists can credit hundreds of unique artists, and
// checking all of them one page-load at a time would mean hammering JOYSOUND
// with a long, bursty sequence of requests on every match request we serve.
// CheckAvailability reports how many artists it actually checked, so callers
// can tell a complete check from a truncated one rather than being misled by
// a short result list.
const maxArtistsPerCheck = 50

// Service answers whether artists are available on karaoke platforms.
type Service struct {
	joysound catalog
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

// CheckAvailability reports JOYSOUND availability for each of the given
// artists, in the order given, up to maxArtistsPerCheck of them. Lookups run
// one at a time with a pause between them — slower than firing them
// concurrently, but considerate of JOYSOUND's servers for what is, from
// their side, a sequence of individual page loads.
func (s *Service) CheckAvailability(ctx context.Context, artists []string) ([]AvailabilityResult, error) {
	artists = boundArtists(artists)

	results := make([]AvailabilityResult, 0, len(artists))
	for i, artist := range artists {
		if i > 0 {
			time.Sleep(pauseBetweenSearches)
		}

		available, err := s.IsArtistAvailable(ctx, artist)
		if err != nil {
			return nil, err
		}
		results = append(results, AvailabilityResult{Artist: artist, Available: available})
	}
	return results, nil
}

// boundArtists truncates the given artists to maxArtistsPerCheck, preserving
// order, so a single check never exceeds that limit.
func boundArtists(artists []string) []string {
	if len(artists) > maxArtistsPerCheck {
		return artists[:maxArtistsPerCheck]
	}
	return artists
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
