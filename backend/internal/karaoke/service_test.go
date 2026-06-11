package karaoke

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
)

// fakeCatalog substitutes for joysoundClient in tests, letting us exercise
// Service's matching/aggregation logic without making live requests to
// JOYSOUND.
type fakeCatalog struct {
	artists map[string][]Artist
	songs   map[string][]Song
	err     error
}

func (f fakeCatalog) searchCross(_ context.Context, keyword string) ([]Song, []Artist, error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	return f.songs[keyword], f.artists[keyword], nil
}

// noopLimiter substitutes for rateLimiter in tests, letting multi-artist
// scenarios run at full speed instead of paying for JOYSOUND's real pacing —
// the pacing behavior itself belongs to rateLimiter, not to Service's
// aggregation logic.
type noopLimiter struct{}

func (noopLimiter) wait() {}

// fakeCache substitutes for postgresCache in tests, letting us exercise
// Service's cache hit/miss/staleness decisions without a real database.
// Service always looks artists up under catalogName, so the fake keys its
// records by artist alone. lookupBatchCalls counts lookupBatch invocations —
// CheckTrackAvailability is expected to make exactly one per call, batching
// every artist into a single round trip rather than looking each one up
// alone.
type fakeCache struct {
	entries          map[string]cacheEntry
	stored           map[string]cacheEntry
	lookupBatchCalls int

	songEntries          map[Track]songCacheEntry
	storedSongs          map[Track]songCacheEntry
	lookupSongBatchCalls int
}

func (f *fakeCache) lookupBatch(_ context.Context, _ string, artists []string) (map[string]cacheEntry, error) {
	f.lookupBatchCalls++

	found := make(map[string]cacheEntry)
	for _, artist := range artists {
		if entry, ok := f.entries[artist]; ok {
			found[artist] = entry
		}
	}
	return found, nil
}

func (f *fakeCache) store(_ context.Context, _, artist string, entry cacheEntry) error {
	if f.stored == nil {
		f.stored = make(map[string]cacheEntry)
	}
	f.stored[artist] = entry
	return nil
}

func (f *fakeCache) lookupSongBatch(_ context.Context, _ string, tracks []Track) (map[Track]songCacheEntry, error) {
	f.lookupSongBatchCalls++

	found := make(map[Track]songCacheEntry)
	for _, track := range tracks {
		if entry, ok := f.songEntries[track]; ok {
			found[track] = entry
		}
	}
	return found, nil
}

func (f *fakeCache) storeSong(_ context.Context, _ string, track Track, entry songCacheEntry) error {
	if f.storedSongs == nil {
		f.storedSongs = make(map[Track]songCacheEntry)
	}
	f.storedSongs[track] = entry
	return nil
}

func TestFindArtistMatching(t *testing.T) {
	tests := []struct {
		name    string
		artists []Artist
		query   string
		want    Artist
		wantOK  bool
	}{
		{
			name:    "no results",
			artists: nil,
			query:   "Bring Me The Horizon",
			wantOK:  false,
		},
		{
			name: "exact match",
			artists: []Artist{
				{ID: "62831", Name: "Bring Me The Horizon"},
			},
			query:  "Bring Me The Horizon",
			want:   Artist{ID: "62831", Name: "Bring Me The Horizon"},
			wantOK: true,
		},
		{
			name: "case-insensitive match",
			artists: []Artist{
				{ID: "62831", Name: "bring me the horizon"},
			},
			query:  "Bring Me The Horizon",
			want:   Artist{ID: "62831", Name: "bring me the horizon"},
			wantOK: true,
		},
		{
			name: "different artist of the same general search is not a match",
			artists: []Artist{
				{ID: "423215", Name: "MACHINE GUN KELLY"},
			},
			query:  "Bring Me The Horizon",
			wantOK: false,
		},
		{
			name: "match present among unrelated results",
			artists: []Artist{
				{ID: "10585", Name: "Def Tech"},
				{ID: "62831", Name: "Bring Me The Horizon"},
			},
			query:  "Bring Me The Horizon",
			want:   Artist{ID: "62831", Name: "Bring Me The Horizon"},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := findArtistMatching(tt.artists, tt.query)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("got (%v, %v), want (%v, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestFindSongMatching(t *testing.T) {
	tests := []struct {
		name   string
		songs  []Song
		title  string
		artist string
		want   Song
		wantOK bool
	}{
		{
			name:   "no results",
			songs:  nil,
			title:  "Off The Edge feat.WISE",
			artist: "Def Tech",
			wantOK: false,
		},
		{
			name: "exact match",
			songs: []Song{
				{ID: "82077", Title: "Off The Edge feat.WISE", Artist: "Def Tech"},
			},
			title:  "Off The Edge feat.WISE",
			artist: "Def Tech",
			want:   Song{ID: "82077", Title: "Off The Edge feat.WISE", Artist: "Def Tech"},
			wantOK: true,
		},
		{
			name: "case-insensitive match",
			songs: []Song{
				{ID: "82077", Title: "off the edge feat.wise", Artist: "def tech"},
			},
			title:  "Off The Edge feat.WISE",
			artist: "Def Tech",
			want:   Song{ID: "82077", Title: "off the edge feat.wise", Artist: "def tech"},
			wantOK: true,
		},
		{
			name: "title matches but artist doesn't",
			songs: []Song{
				{ID: "920844", Title: "maybe feat. Bring Me The Horizon", Artist: "MACHINE GUN KELLY"},
			},
			title:  "maybe feat. Bring Me The Horizon",
			artist: "Bring Me The Horizon",
			wantOK: false,
		},
		{
			name: "match present among unrelated results",
			songs: []Song{
				{ID: "920844", Title: "maybe feat. Bring Me The Horizon", Artist: "MACHINE GUN KELLY"},
				{ID: "82077", Title: "Off The Edge feat.WISE", Artist: "Def Tech"},
			},
			title:  "Off The Edge feat.WISE",
			artist: "Def Tech",
			want:   Song{ID: "82077", Title: "Off The Edge feat.WISE", Artist: "Def Tech"},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := findSongMatching(tt.songs, tt.title, tt.artist)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("got (%v, %v), want (%v, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestCheckTrackAvailability(t *testing.T) {
	t.Run("song and artist both matched from one combined search", func(t *testing.T) {
		track := Track{Artist: "Def Tech", Title: "Off The Edge feat.WISE"}
		svc := &Service{
			joysound: fakeCatalog{
				songs: map[string][]Song{
					"Off The Edge feat.WISE": {{ID: "82077", Title: "Off The Edge feat.WISE", Artist: "Def Tech"}},
				},
				artists: map[string][]Artist{
					"Off The Edge feat.WISE": {{ID: "10585", Name: "Def Tech"}},
					// A different ID under the artist-name keyword — if this
					// were used, it would prove the (unwanted) fallback
					// search ran instead of resolving from the title search.
					"Def Tech": {{ID: "99999", Name: "Def Tech"}},
				},
			},
			cache:   &fakeCache{},
			limiter: noopLimiter{},
		}

		got, err := svc.CheckTrackAvailability(context.Background(), []Track{track})
		if err != nil {
			t.Fatalf("CheckTrackAvailability returned error: %v", err)
		}

		want := []TrackAvailabilityResult{
			{
				Artist:            "Def Tech",
				Title:             "Off The Edge feat.WISE",
				SongJoysoundURL:   "https://www.joysound.com/web/search/song/82077",
				ArtistJoysoundURL: "https://www.joysound.com/web/search/artist/10585",
			},
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		storedSong, ok := svc.cache.(*fakeCache).storedSongs[track]
		if !ok || storedSong.CatalogSongID != "82077" {
			t.Errorf("storedSongs[track] = %v, ok=%v, want CatalogSongID 82077", storedSong, ok)
		}
		storedArtist, ok := svc.cache.(*fakeCache).stored["Def Tech"]
		if !ok || storedArtist.CatalogArtistID != "10585" {
			t.Errorf("stored[Def Tech] = %v, ok=%v, want CatalogArtistID 10585 — the artist found on the title-search page, not the fallback search", storedArtist, ok)
		}
	})

	t.Run("song matched, artist not on that page falls back to a name search", func(t *testing.T) {
		track := Track{Artist: "Architects", Title: "Doomed"}
		svc := &Service{
			joysound: fakeCatalog{
				songs:   map[string][]Song{"Doomed": {{ID: "5", Title: "Doomed", Artist: "Architects"}}},
				artists: map[string][]Artist{"Architects": {{ID: "1", Name: "Architects"}}},
			},
			cache:   &fakeCache{},
			limiter: noopLimiter{},
		}

		got, err := svc.CheckTrackAvailability(context.Background(), []Track{track})
		if err != nil {
			t.Fatalf("CheckTrackAvailability returned error: %v", err)
		}

		want := []TrackAvailabilityResult{
			{
				Artist:            "Architects",
				Title:             "Doomed",
				SongJoysoundURL:   "https://www.joysound.com/web/search/song/5",
				ArtistJoysoundURL: "https://www.joysound.com/web/search/artist/1",
			},
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("no song match falls back to a name search for the artist link", func(t *testing.T) {
		track := Track{Artist: "Architects", Title: "Animals"}
		svc := &Service{
			joysound: fakeCatalog{
				songs:   map[string][]Song{"Animals": {}},
				artists: map[string][]Artist{"Architects": {{ID: "1", Name: "Architects"}}},
			},
			cache:   &fakeCache{},
			limiter: noopLimiter{},
		}

		got, err := svc.CheckTrackAvailability(context.Background(), []Track{track})
		if err != nil {
			t.Fatalf("CheckTrackAvailability returned error: %v", err)
		}

		want := []TrackAvailabilityResult{
			{Artist: "Architects", Title: "Animals", ArtistJoysoundURL: "https://www.joysound.com/web/search/artist/1"},
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		stored, ok := svc.cache.(*fakeCache).storedSongs[track]
		if !ok {
			t.Fatal("expected song entry to be cached")
		}
		if stored.CatalogSongID != "" {
			t.Errorf("cached CatalogSongID = %q, want empty (no song match)", stored.CatalogSongID)
		}
	})

	t.Run("no match anywhere", func(t *testing.T) {
		track := Track{Artist: "Thornhill", Title: "Discipline"}
		svc := &Service{
			joysound: fakeCatalog{
				songs:   map[string][]Song{"Discipline": {}},
				artists: map[string][]Artist{"Discipline": {}, "Thornhill": {}},
			},
			cache:   &fakeCache{},
			limiter: noopLimiter{},
		}

		got, err := svc.CheckTrackAvailability(context.Background(), []Track{track})
		if err != nil {
			t.Fatalf("CheckTrackAvailability returned error: %v", err)
		}

		want := []TrackAvailabilityResult{
			{Artist: "Thornhill", Title: "Discipline"},
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("fresh cache hits for both song and artist are free", func(t *testing.T) {
		track := Track{Artist: "Def Tech", Title: "Off The Edge feat.WISE"}
		cache := &fakeCache{
			songEntries: map[Track]songCacheEntry{
				track: {Available: true, CatalogSongID: "82077", LastChecked: time.Now()},
			},
			entries: map[string]cacheEntry{
				"Def Tech": {Available: true, CatalogArtistID: "10585", LastChecked: time.Now()},
			},
		}
		svc := &Service{joysound: fakeCatalog{err: errors.New("should not be called")}, cache: cache, limiter: noopLimiter{}}

		got, err := svc.CheckTrackAvailability(context.Background(), []Track{track})
		if err != nil {
			t.Fatalf("CheckTrackAvailability returned error: %v", err)
		}

		want := []TrackAvailabilityResult{
			{
				Artist:            "Def Tech",
				Title:             "Off The Edge feat.WISE",
				SongJoysoundURL:   "https://www.joysound.com/web/search/song/82077",
				ArtistJoysoundURL: "https://www.joysound.com/web/search/artist/10585",
			},
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("fresh song cache hit with stale artist cache resolves the artist without a title search", func(t *testing.T) {
		track := Track{Artist: "Architects", Title: "Animals"}
		cache := &fakeCache{
			songEntries: map[Track]songCacheEntry{
				track: {Available: false, CatalogSongID: "", LastChecked: time.Now()},
			},
		}
		svc := &Service{
			joysound: fakeCatalog{artists: map[string][]Artist{"Architects": {{ID: "1", Name: "Architects"}}}},
			cache:    cache,
			limiter:  noopLimiter{},
		}

		got, err := svc.CheckTrackAvailability(context.Background(), []Track{track})
		if err != nil {
			t.Fatalf("CheckTrackAvailability returned error: %v", err)
		}

		want := []TrackAvailabilityResult{
			{Artist: "Architects", Title: "Animals", ArtistJoysoundURL: "https://www.joysound.com/web/search/artist/1"},
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("batches every track's cache lookups into a single round trip", func(t *testing.T) {
		tracks := []Track{
			{Artist: "Def Tech", Title: "Off The Edge feat.WISE"},
			{Artist: "Architects", Title: "Animals"},
		}
		cache := &fakeCache{}
		svc := &Service{
			joysound: fakeCatalog{
				songs: map[string][]Song{"Off The Edge feat.WISE": {}, "Animals": {}},
				artists: map[string][]Artist{
					"Off The Edge feat.WISE": {}, "Animals": {},
					"Def Tech": {}, "Architects": {},
				},
			},
			cache:   cache,
			limiter: noopLimiter{},
		}

		if _, err := svc.CheckTrackAvailability(context.Background(), tracks); err != nil {
			t.Fatalf("CheckTrackAvailability returned error: %v", err)
		}

		if cache.lookupSongBatchCalls != 1 {
			t.Errorf("got %d lookupSongBatch calls, want exactly 1", cache.lookupSongBatchCalls)
		}
		if cache.lookupBatchCalls != 1 {
			t.Errorf("got %d lookupBatch calls, want exactly 1", cache.lookupBatchCalls)
		}
	})

	t.Run("propagates a search failure", func(t *testing.T) {
		svc := &Service{joysound: fakeCatalog{err: errors.New("boom")}, cache: &fakeCache{}, limiter: noopLimiter{}}

		if _, err := svc.CheckTrackAvailability(context.Background(), []Track{{Artist: "Architects", Title: "Animals"}}); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("stops once the live-search budget is exhausted", func(t *testing.T) {
		tracks := make([]Track, maxLiveSearchesPerCheck)
		for i := range tracks {
			tracks[i] = Track{Artist: fmt.Sprintf("Artist %d", i), Title: fmt.Sprintf("Song %d", i)}
		}

		svc := &Service{joysound: fakeCatalog{}, cache: &fakeCache{}, limiter: noopLimiter{}}

		got, err := svc.CheckTrackAvailability(context.Background(), tracks)
		if err != nil {
			t.Fatalf("CheckTrackAvailability returned error: %v", err)
		}

		// Each track with no song match and no artist match on the title
		// search costs 2 live searches (title search + artist name search),
		// so only half the budget's worth of tracks can be fully resolved.
		want := maxLiveSearchesPerCheck / 2
		if len(got) != want {
			t.Fatalf("got %d results, want %d — each unresolved track costs 2 live searches", len(got), want)
		}
		for i, result := range got {
			if result.Artist != tracks[i].Artist || result.Title != tracks[i].Title {
				t.Errorf("result %d: got %q/%q, want %q/%q — results should be an in-order prefix", i, result.Artist, result.Title, tracks[i].Artist, tracks[i].Title)
			}
		}
	})
}
