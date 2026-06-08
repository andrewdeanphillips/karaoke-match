package karaoke

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// catalog is the subset of joysoundClient that Service depends on — narrow
// enough that tests can substitute a fake and exercise matching/aggregation
// logic without making live requests to JOYSOUND.
type catalog interface {
	search(ctx context.Context, keyword string) ([]Artist, error)
}

// pauseBetweenSearches is the minimum spacing rateLimiter enforces between
// live JOYSOUND requests — considerate of their servers, since this is a
// sequence of individual page loads, not a bulk API.
const pauseBetweenSearches = 200 * time.Millisecond

// limiter is the subset of rateLimiter that Service depends on — narrow
// enough that tests can substitute a no-op and exercise multi-artist
// aggregation logic without paying for JOYSOUND's real pacing.
type limiter interface {
	wait()
}

// rateLimiter enforces at least pauseBetweenSearches between live JOYSOUND
// requests, based on real elapsed time rather than a fixed pause before every
// call — so the very first call, with nothing yet to space away from, never
// waits.
type rateLimiter struct {
	mu   sync.Mutex
	last time.Time
}

func (r *rateLimiter) wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if since := time.Since(r.last); since < pauseBetweenSearches {
		time.Sleep(pauseBetweenSearches - since)
	}
	r.last = time.Now()
}

// maxLiveSearchesPerCheck bounds how many live JOYSOUND searches a single
// CheckAvailability call will make. Cache hits are free and don't count
// against it — only artists that actually require a live lookup spend from
// this budget, so a well-cached playlist can be checked more completely than
// a cold one for the same JOYSOUND cost.
const maxLiveSearchesPerCheck = 50

// catalogName identifies JOYSOUND in cached records. Service is JOYSOUND-
// specific today, but the cache's schema models the more general
// (artist, catalog) shape so a future second catalog wouldn't require a
// migration — just another constant like this one.
const catalogName = "joysound"

// cacheTTL is how long a cached availability record is trusted before it's
// treated as stale and re-checked live. Karaoke catalogs don't churn often,
// so a long TTL trades a little staleness for far fewer repeat lookups.
const cacheTTL = 30 * 24 * time.Hour

// Service answers whether artists are available on karaoke platforms.
type Service struct {
	joysound catalog
	cache    cache
	limiter  limiter
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{joysound: newJoysoundClient(), cache: newPostgresCache(pool), limiter: &rateLimiter{}}
}

// IsArtistAvailable reports whether JOYSOUND lists the given artist. A fresh
// cached result is returned as-is, with no request to JOYSOUND involved at
// all; otherwise it searches live and caches what it finds for next time.
func (s *Service) IsArtistAvailable(ctx context.Context, artist string) (bool, error) {
	if entry, ok := s.freshCacheHit(ctx, artist); ok {
		return entry.Available, nil
	}
	return s.searchLive(ctx, artist)
}

// freshCacheHit returns the cached availability record for the given artist,
// if one exists and is still within cacheTTL. A lookup failure is logged and
// treated as a miss — caching is a performance optimization, not a
// correctness requirement, so a flaky cache should degrade the feature to
// "a bit slower," never "broken."
func (s *Service) freshCacheHit(ctx context.Context, artist string) (cacheEntry, bool) {
	entry, found, err := s.cache.lookup(ctx, catalogName, artist)
	if err != nil {
		log.Printf("checking cache for artist %q: %v", artist, err)
		return cacheEntry{}, false
	}
	return entry, found && time.Since(entry.LastChecked) < cacheTTL
}

// searchLive paces itself against JOYSOUND's servers, searches live for the
// given artist, and caches what it finds for next time (best-effort — a
// caching failure is logged rather than failing the lookup, for the same
// reason a cache-read failure is).
//
// JOYSOUND's search-results page itself separates artist matches from song
// matches, so every result search returns is already an artist — we just
// need to confirm one of them is actually the artist we're looking for,
// rather than a same-named act in JOYSOUND's catalog.
func (s *Service) searchLive(ctx context.Context, artist string) (bool, error) {
	s.limiter.wait()

	artists, err := s.joysound.search(ctx, artist)
	if err != nil {
		return false, fmt.Errorf("searching JOYSOUND for artist %q: %w", artist, err)
	}

	match, available := findArtistNamed(artists, artist)
	entry := cacheEntry{Available: available, CatalogArtistID: match.ID, LastChecked: time.Now()}
	if err := s.cache.store(ctx, catalogName, artist, entry); err != nil {
		log.Printf("caching availability for artist %q: %v", artist, err)
	}

	return available, nil
}

// CheckAvailability reports JOYSOUND availability for each of the given
// artists, in the order given, stopping early if answering the next one would
// exceed maxLiveSearchesPerCheck live JOYSOUND searches. Artists with a fresh
// cached result are free — only artists that actually require a live search
// count against that limit, so a well-cached playlist can yield more results
// than a cold one would, for the same JOYSOUND cost.
func (s *Service) CheckAvailability(ctx context.Context, artists []string) ([]AvailabilityResult, error) {
	results := make([]AvailabilityResult, 0, len(artists))
	liveSearches := 0

	for _, artist := range artists {
		entry, ok := s.freshCacheHit(ctx, artist)
		if ok {
			results = append(results, AvailabilityResult{Artist: artist, Available: entry.Available})
			continue
		}

		if liveSearches >= maxLiveSearchesPerCheck {
			break
		}

		available, err := s.searchLive(ctx, artist)
		if err != nil {
			return nil, err
		}
		liveSearches++
		results = append(results, AvailabilityResult{Artist: artist, Available: available})
	}
	return results, nil
}

// findArtistNamed returns the first result whose name matches the given
// name, ignoring case to tolerate capitalization differences.
func findArtistNamed(artists []Artist, name string) (Artist, bool) {
	for _, a := range artists {
		if strings.EqualFold(a.Name, name) {
			return a, true
		}
	}
	return Artist{}, false
}
