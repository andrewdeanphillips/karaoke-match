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
	searchSongs(ctx context.Context, keyword string) ([]Song, error)
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

// joysoundArtistURL returns the direct link to an artist's JOYSOUND page.
// id is JOYSOUND's own stable numeric identifier (e.g. "62831").
func joysoundArtistURL(id string) string {
	return "https://www.joysound.com/web/search/artist/" + id
}

// joysoundSongURL returns the direct link to a song's JOYSOUND page. id is
// JOYSOUND's own stable numeric identifier (e.g. "82077").
func joysoundSongURL(id string) string {
	return "https://www.joysound.com/web/search/song/" + id
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
func (s *Service) searchLive(ctx context.Context, artist string) (cacheEntry, error) {
	s.limiter.wait()

	artists, err := s.joysound.search(ctx, artist)
	if err != nil {
		return cacheEntry{}, fmt.Errorf("searching JOYSOUND for artist %q: %w", artist, err)
	}

	match, available := findArtistNamed(artists, artist)
	entry := cacheEntry{Available: available, CatalogArtistID: match.ID, LastChecked: time.Now()}
	if err := s.cache.store(ctx, catalogName, artist, entry); err != nil {
		log.Printf("caching availability for artist %q: %v", artist, err)
	}

	return entry, nil
}

// resultFromEntry builds an AvailabilityResult from a cache entry, including
// the JOYSOUND artist URL when the entry has a catalog ID.
func resultFromEntry(artist string, entry cacheEntry) AvailabilityResult {
	result := AvailabilityResult{Artist: artist, Available: entry.Available}
	if entry.CatalogArtistID != "" {
		result.JoysoundURL = joysoundArtistURL(entry.CatalogArtistID)
	}
	return result
}

// CheckAvailability reports JOYSOUND availability for each of the given
// artists, in the order given, stopping early if answering the next one would
// exceed maxLiveSearchesPerCheck live JOYSOUND searches. Artists with a fresh
// cached result are free — only artists that actually require a live search
// count against that limit, so a well-cached playlist can yield more results
// than a cold one would, for the same JOYSOUND cost.
//
// Cache lookups for every artist are batched into one round trip up front
// (rather than one round trip per artist) — the playlists this checks can
// have dozens of unique artists, and against a remote database each round
// trip costs real network latency.
func (s *Service) CheckAvailability(ctx context.Context, artists []string) ([]AvailabilityResult, error) {
	cached, err := s.cache.lookupBatch(ctx, catalogName, artists)
	if err != nil {
		log.Printf("batch-checking cache for %d artists: %v", len(artists), err)
		cached = nil
	}

	results := make([]AvailabilityResult, 0, len(artists))
	liveSearches := 0

	for _, artist := range artists {
		if entry, ok := cached[artist]; ok && time.Since(entry.LastChecked) < cacheTTL {
			results = append(results, resultFromEntry(artist, entry))
			continue
		}

		if liveSearches >= maxLiveSearchesPerCheck {
			break
		}

		entry, err := s.searchLive(ctx, artist)
		if err != nil {
			return nil, err
		}
		liveSearches++
		results = append(results, resultFromEntry(artist, entry))
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

// findSongMatching returns the first result whose title and artist both
// match the given track, ignoring case. Matching on title alone isn't
// enough — JOYSOUND's title search can return same-named songs by other
// artists — so both fields must agree.
func findSongMatching(songs []Song, title, artist string) (Song, bool) {
	for _, s := range songs {
		if strings.EqualFold(s.Title, title) && strings.EqualFold(s.Artist, artist) {
			return s, true
		}
	}
	return Song{}, false
}

// resultFromSongEntry builds a TrackAvailabilityResult from a cached song
// entry. JoysoundURL points at the matched song's page when one was found;
// otherwise, if the track's artist is available, it falls back to the
// artist's page using cachedArtists.
func resultFromSongEntry(track Track, entry songCacheEntry, cachedArtists map[string]cacheEntry) TrackAvailabilityResult {
	result := TrackAvailabilityResult{Artist: track.Artist, Title: track.Title, Available: entry.Available}
	switch {
	case entry.CatalogSongID != "":
		result.JoysoundURL = joysoundSongURL(entry.CatalogSongID)
	case entry.Available:
		if artistEntry, ok := cachedArtists[track.Artist]; ok && artistEntry.CatalogArtistID != "" {
			result.JoysoundURL = joysoundArtistURL(artistEntry.CatalogArtistID)
		}
	}
	return result
}

// artistAvailable reports whether a track's artist is available on JOYSOUND,
// preferring a fresh cached entry and otherwise spending one live search (if
// budget allows). On a live search, cachedArtists is updated in place so a
// later call to resultFromSongEntry for the same artist can find its
// catalog ID for the fallback URL.
//
// If budget is exhausted with no fresh cache entry, this reports
// unavailable without searching — the same approximation CheckAvailability
// makes when its own budget runs out, just at a finer grain. The track's
// song-cache entry will simply be re-checked after cacheTTL.
func (s *Service) artistAvailable(ctx context.Context, artist string, cachedArtists map[string]cacheEntry, budget int) (bool, int, error) {
	if entry, ok := cachedArtists[artist]; ok && time.Since(entry.LastChecked) < cacheTTL {
		return entry.Available, 0, nil
	}

	if budget < 1 {
		return false, 0, nil
	}

	entry, err := s.searchLive(ctx, artist)
	if err != nil {
		return false, 1, err
	}
	cachedArtists[artist] = entry
	return entry.Available, 1, nil
}

// searchSongLive paces itself against JOYSOUND's servers, searches live for
// the given track by title, and caches what it finds for next time. If no
// song result matches both the track's title and artist, it falls back to
// artistAvailable so the track still gets a useful result when JOYSOUND has
// the artist but not this specific song.
//
// It returns the cache entry to build a result from, and how many live
// JOYSOUND searches it performed (1 for the song search, plus 1 more if the
// artist fallback also required a live search).
func (s *Service) searchSongLive(ctx context.Context, track Track, cachedArtists map[string]cacheEntry, budget int) (songCacheEntry, int, error) {
	s.limiter.wait()

	songs, err := s.joysound.searchSongs(ctx, track.Title)
	if err != nil {
		return songCacheEntry{}, 1, fmt.Errorf("searching JOYSOUND for song %q by %q: %w", track.Title, track.Artist, err)
	}

	if match, ok := findSongMatching(songs, track.Title, track.Artist); ok {
		entry := songCacheEntry{Available: true, CatalogSongID: match.ID, LastChecked: time.Now()}
		s.storeSongEntry(ctx, track, entry)
		return entry, 1, nil
	}

	available, searchesUsed, err := s.artistAvailable(ctx, track.Artist, cachedArtists, budget-1)
	if err != nil {
		return songCacheEntry{}, 1 + searchesUsed, err
	}

	entry := songCacheEntry{Available: available, LastChecked: time.Now()}
	s.storeSongEntry(ctx, track, entry)
	return entry, 1 + searchesUsed, nil
}

// storeSongEntry caches a track's availability (best-effort — a caching
// failure is logged rather than failing the lookup, for the same reason
// searchLive's caching failure is).
func (s *Service) storeSongEntry(ctx context.Context, track Track, entry songCacheEntry) {
	if err := s.cache.storeSong(ctx, catalogName, track, entry); err != nil {
		log.Printf("caching song availability for %q by %q: %v", track.Title, track.Artist, err)
	}
}

// CheckTrackAvailability reports JOYSOUND availability for each of the given
// tracks, in the order given, stopping early if answering the next one would
// exceed maxLiveSearchesPerCheck live JOYSOUND searches. It mirrors
// CheckAvailability's caching and budget behavior at song granularity: a
// fresh cached result is free, and cache lookups for every track (and every
// track's artist, for the fallback path) are batched into one round trip
// each up front.
func (s *Service) CheckTrackAvailability(ctx context.Context, tracks []Track) ([]TrackAvailabilityResult, error) {
	cachedSongs, err := s.cache.lookupSongBatch(ctx, catalogName, tracks)
	if err != nil {
		log.Printf("batch-checking song cache for %d tracks: %v", len(tracks), err)
		cachedSongs = nil
	}

	artists := make([]string, len(tracks))
	for i, track := range tracks {
		artists[i] = track.Artist
	}
	cachedArtists, err := s.cache.lookupBatch(ctx, catalogName, artists)
	if err != nil {
		log.Printf("batch-checking artist cache for %d artists: %v", len(artists), err)
		cachedArtists = nil
	}
	if cachedArtists == nil {
		cachedArtists = make(map[string]cacheEntry)
	}

	results := make([]TrackAvailabilityResult, 0, len(tracks))
	liveSearches := 0

	for _, track := range tracks {
		if entry, ok := cachedSongs[track]; ok && time.Since(entry.LastChecked) < cacheTTL {
			results = append(results, resultFromSongEntry(track, entry, cachedArtists))
			continue
		}

		if liveSearches >= maxLiveSearchesPerCheck {
			break
		}

		entry, searches, err := s.searchSongLive(ctx, track, cachedArtists, maxLiveSearchesPerCheck-liveSearches)
		if err != nil {
			return nil, err
		}
		liveSearches += searches
		results = append(results, resultFromSongEntry(track, entry, cachedArtists))
	}
	return results, nil
}
