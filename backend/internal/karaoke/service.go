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
	searchCross(ctx context.Context, keyword string) ([]Song, []Artist, error)
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
// CheckTrackAvailability call will make. Cache hits are free and don't count
// against it — only tracks/artists that actually require a live lookup spend
// from this budget, so a well-cached playlist can be checked more completely
// than a cold one for the same JOYSOUND cost.
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
// given artist by name, and caches what it finds for next time (best-effort
// — a caching failure is logged rather than failing the lookup, for the same
// reason a cache-read failure is).
//
// JOYSOUND's search-results page itself separates artist matches from song
// matches, so every artist result a search returns is already an artist — we
// just need to confirm one of them is actually the artist we're looking for,
// rather than a same-named act in JOYSOUND's catalog.
func (s *Service) searchLive(ctx context.Context, artist string) (cacheEntry, error) {
	s.limiter.wait()

	_, artists, err := s.joysound.searchCross(ctx, artist)
	if err != nil {
		return cacheEntry{}, fmt.Errorf("searching JOYSOUND for artist %q: %w", artist, err)
	}

	match, ok := findArtistMatching(artists, artist)
	entry := cacheEntry{Available: ok, CatalogArtistID: match.ID, LastChecked: time.Now()}
	if err := s.cache.store(ctx, catalogName, artist, entry); err != nil {
		log.Printf("caching availability for artist %q: %v", artist, err)
	}

	return entry, nil
}

// findArtistMatching returns the first result whose name matches the given
// name, ignoring case to tolerate capitalization differences.
func findArtistMatching(artists []Artist, name string) (Artist, bool) {
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

// buildResult assembles a TrackAvailabilityResult from a track's resolved
// song and artist cache entries. Either URL is present only when its entry
// has a catalog ID — the two are independent, so a track can have a song
// link, an artist link, both, or neither.
func buildResult(track Track, songEntry songCacheEntry, artistEntry cacheEntry) TrackAvailabilityResult {
	result := TrackAvailabilityResult{Artist: track.Artist, Title: track.Title}
	if songEntry.CatalogSongID != "" {
		result.SongJoysoundURL = joysoundSongURL(songEntry.CatalogSongID)
	}
	if artistEntry.CatalogArtistID != "" {
		result.ArtistJoysoundURL = joysoundArtistURL(artistEntry.CatalogArtistID)
	}
	return result
}

// freshSongEntry returns a track's cached song entry if one exists and is
// within cacheTTL.
func freshSongEntry(cached map[Track]songCacheEntry, track Track) (songCacheEntry, bool) {
	entry, ok := cached[track]
	if !ok || time.Since(entry.LastChecked) >= cacheTTL {
		return songCacheEntry{}, false
	}
	return entry, true
}

// freshArtistEntry returns an artist's cached entry if one exists and is
// within cacheTTL.
func freshArtistEntry(cached map[string]cacheEntry, artist string) (cacheEntry, bool) {
	entry, ok := cached[artist]
	if !ok || time.Since(entry.LastChecked) >= cacheTTL {
		return cacheEntry{}, false
	}
	return entry, true
}

// storeSongEntry caches a track's song availability (best-effort — a caching
// failure is logged rather than failing the lookup, for the same reason
// searchLive's caching failure is).
func (s *Service) storeSongEntry(ctx context.Context, track Track, entry songCacheEntry) {
	if err := s.cache.storeSong(ctx, catalogName, track, entry); err != nil {
		log.Printf("caching song availability for %q by %q: %v", track.Title, track.Artist, err)
	}
}

// resolveTrack resolves whichever of a track's song and artist entries
// aren't already fresh, spending up to budget live JOYSOUND searches, and
// returns the resolved entries plus how many searches it performed.
//
// If the song entry isn't fresh, it runs one combined search keyed on the
// track's title — JOYSOUND's cross-search returns both song and artist
// results in the same response, so if the artist entry also isn't fresh, it
// may be resolved from that same page for free. Only if the artist still
// isn't resolved (and budget remains) does this spend a second search keyed
// on the artist's name, mirroring the matching searchLive already does for
// artist-only lookups.
func (s *Service) resolveTrack(ctx context.Context, track Track, songEntry songCacheEntry, artistEntry cacheEntry, songFresh, artistFresh bool, cachedArtists map[string]cacheEntry, budget int) (songCacheEntry, cacheEntry, int, error) {
	searches := 0

	if !songFresh {
		s.limiter.wait()

		songs, artists, err := s.joysound.searchCross(ctx, track.Title)
		if err != nil {
			return songCacheEntry{}, cacheEntry{}, searches, fmt.Errorf("searching JOYSOUND for %q by %q: %w", track.Title, track.Artist, err)
		}
		searches++

		songEntry = songCacheEntry{LastChecked: time.Now()}
		if match, ok := findSongMatching(songs, track.Title, track.Artist); ok {
			songEntry.Available = true
			songEntry.CatalogSongID = match.ID
		}
		s.storeSongEntry(ctx, track, songEntry)

		if !artistFresh {
			if match, ok := findArtistMatching(artists, track.Artist); ok {
				artistEntry = cacheEntry{Available: true, CatalogArtistID: match.ID, LastChecked: time.Now()}
				if err := s.cache.store(ctx, catalogName, track.Artist, artistEntry); err != nil {
					log.Printf("caching availability for artist %q: %v", track.Artist, err)
				}
				cachedArtists[track.Artist] = artistEntry
				artistFresh = true
			}
		}
	}

	if !artistFresh {
		if searches >= budget {
			return songEntry, artistEntry, searches, nil
		}

		entry, err := s.searchLive(ctx, track.Artist)
		if err != nil {
			return songEntry, artistEntry, searches, err
		}
		searches++
		cachedArtists[track.Artist] = entry
		artistEntry = entry
	}

	return songEntry, artistEntry, searches, nil
}

// CheckTrackAvailability reports JOYSOUND links for each of the given
// tracks, in the order given, stopping early if resolving the next one could
// exceed maxLiveSearchesPerCheck live JOYSOUND searches. A track whose song
// and artist entries are both fresh in the cache is free; otherwise
// resolveTrack spends up to two live searches on it. Cache lookups for every
// track (and every track's artist) are batched into one round trip each up
// front.
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
		songEntry, songFresh := freshSongEntry(cachedSongs, track)
		artistEntry, artistFresh := freshArtistEntry(cachedArtists, track.Artist)

		if !songFresh || !artistFresh {
			if liveSearches >= maxLiveSearchesPerCheck {
				break
			}

			var searches int
			songEntry, artistEntry, searches, err = s.resolveTrack(ctx, track, songEntry, artistEntry, songFresh, artistFresh, cachedArtists, maxLiveSearchesPerCheck-liveSearches)
			if err != nil {
				return nil, err
			}
			liveSearches += searches
		}

		results = append(results, buildResult(track, songEntry, artistEntry))
	}
	return results, nil
}
