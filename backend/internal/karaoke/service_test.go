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
	err     error
}

func (f fakeCatalog) search(_ context.Context, keyword string) ([]Artist, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.artists[keyword], nil
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
// records by artist alone.
type fakeCache struct {
	entries map[string]cacheEntry
	stored  map[string]cacheEntry
}

func (f *fakeCache) lookup(_ context.Context, _, artist string) (cacheEntry, bool, error) {
	entry, found := f.entries[artist]
	return entry, found, nil
}

func (f *fakeCache) store(_ context.Context, _, artist string, entry cacheEntry) error {
	if f.stored == nil {
		f.stored = make(map[string]cacheEntry)
	}
	f.stored[artist] = entry
	return nil
}

func TestFindArtistNamed(t *testing.T) {
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
			got, ok := findArtistNamed(tt.artists, tt.query)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("got (%v, %v), want (%v, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestIsArtistAvailable(t *testing.T) {
	t.Run("fresh cache hit is returned without searching", func(t *testing.T) {
		cache := &fakeCache{entries: map[string]cacheEntry{
			"Architects": {Available: true, CatalogArtistID: "1", LastChecked: time.Now()},
		}}
		svc := &Service{joysound: fakeCatalog{err: errors.New("search should not run on a fresh cache hit")}, cache: cache, limiter: noopLimiter{}}

		got, err := svc.IsArtistAvailable(context.Background(), "Architects")
		if err != nil {
			t.Fatalf("IsArtistAvailable returned error: %v", err)
		}
		if !got {
			t.Errorf("got %v, want true", got)
		}
	})

	t.Run("cache miss searches live and stores the result", func(t *testing.T) {
		cache := &fakeCache{}
		svc := &Service{
			joysound: fakeCatalog{artists: map[string][]Artist{
				"Architects": {{ID: "1", Name: "Architects"}},
			}},
			cache:   cache,
			limiter: noopLimiter{},
		}

		got, err := svc.IsArtistAvailable(context.Background(), "Architects")
		if err != nil {
			t.Fatalf("IsArtistAvailable returned error: %v", err)
		}
		if !got {
			t.Errorf("got %v, want true", got)
		}

		stored, ok := cache.stored["Architects"]
		if !ok {
			t.Fatal("expected the live result to be cached")
		}
		if !stored.Available || stored.CatalogArtistID != "1" {
			t.Errorf("got stored entry %+v, want available with catalog artist ID %q", stored, "1")
		}
	})

	t.Run("stale cache entry is refreshed with a live search", func(t *testing.T) {
		cache := &fakeCache{entries: map[string]cacheEntry{
			"Architects": {Available: false, LastChecked: time.Now().Add(-(cacheTTL + time.Hour))},
		}}
		svc := &Service{
			joysound: fakeCatalog{artists: map[string][]Artist{
				"Architects": {{ID: "1", Name: "Architects"}},
			}},
			cache:   cache,
			limiter: noopLimiter{},
		}

		got, err := svc.IsArtistAvailable(context.Background(), "Architects")
		if err != nil {
			t.Fatalf("IsArtistAvailable returned error: %v", err)
		}
		if !got {
			t.Errorf("got %v, want true — a stale cache entry should be refreshed by a live search", got)
		}

		stored, ok := cache.stored["Architects"]
		if !ok {
			t.Fatal("expected the refreshed result to be cached")
		}
		if !stored.Available {
			t.Errorf("got stored entry %+v, want it refreshed to available", stored)
		}
	})
}

func TestCheckAvailability(t *testing.T) {
	t.Run("aggregates a result per artist, in order", func(t *testing.T) {
		svc := &Service{
			joysound: fakeCatalog{artists: map[string][]Artist{
				"Bring Me The Horizon": {{ID: "62831", Name: "Bring Me The Horizon"}},
				"Architects":           {{ID: "1", Name: "Architects"}},
			}},
			cache:   &fakeCache{},
			limiter: noopLimiter{},
		}

		got, err := svc.CheckAvailability(context.Background(), []string{"Bring Me The Horizon", "Thornhill", "Architects"})
		if err != nil {
			t.Fatalf("CheckAvailability returned error: %v", err)
		}

		want := []AvailabilityResult{
			{Artist: "Bring Me The Horizon", Available: true},
			{Artist: "Thornhill", Available: false},
			{Artist: "Architects", Available: true},
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("propagates a search failure", func(t *testing.T) {
		svc := &Service{joysound: fakeCatalog{err: errors.New("boom")}, cache: &fakeCache{}, limiter: noopLimiter{}}

		if _, err := svc.CheckAvailability(context.Background(), []string{"Architects"}); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("stops once the live-search budget is exhausted", func(t *testing.T) {
		artists := make([]string, maxLiveSearchesPerCheck+10)
		for i := range artists {
			artists[i] = fmt.Sprintf("Artist %d", i)
		}

		svc := &Service{joysound: fakeCatalog{}, cache: &fakeCache{}, limiter: noopLimiter{}}

		got, err := svc.CheckAvailability(context.Background(), artists)
		if err != nil {
			t.Fatalf("CheckAvailability returned error: %v", err)
		}

		if len(got) != maxLiveSearchesPerCheck {
			t.Fatalf("got %d results, want exactly the live-search cap of %d", len(got), maxLiveSearchesPerCheck)
		}
		for i, result := range got {
			if result.Artist != artists[i] {
				t.Errorf("result %d: got artist %q, want %q — results should be an in-order prefix", i, result.Artist, artists[i])
			}
		}
	})

	t.Run("cached hits don't count against the live-search budget, so a well-cached batch can return more than the cap", func(t *testing.T) {
		artists := make([]string, maxLiveSearchesPerCheck+10)
		for i := range artists {
			artists[i] = fmt.Sprintf("Artist %d", i)
		}

		const cachedCount = 20
		cachedEntries := make(map[string]cacheEntry, cachedCount)
		for _, artist := range artists[:cachedCount] {
			cachedEntries[artist] = cacheEntry{Available: true, LastChecked: time.Now()}
		}

		cache := &fakeCache{entries: cachedEntries}
		svc := &Service{joysound: fakeCatalog{}, cache: cache, limiter: noopLimiter{}}

		got, err := svc.CheckAvailability(context.Background(), artists)
		if err != nil {
			t.Fatalf("CheckAvailability returned error: %v", err)
		}

		if len(got) != len(artists) {
			t.Fatalf("got %d results, want all %d artists checked — cached hits shouldn't trigger an early stop", len(got), len(artists))
		}
		if len(got) <= maxLiveSearchesPerCheck {
			t.Fatalf("got %d results, want more than the live-search cap of %d — otherwise this scenario isn't proving what it claims to", len(got), maxLiveSearchesPerCheck)
		}
		if len(cache.stored) > maxLiveSearchesPerCheck {
			t.Errorf("performed %d live searches (one per cache write), want at most %d", len(cache.stored), maxLiveSearchesPerCheck)
		}
	})
}
