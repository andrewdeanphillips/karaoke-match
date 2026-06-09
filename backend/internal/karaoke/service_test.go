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
// records by artist alone. lookupBatchCalls counts lookupBatch invocations —
// CheckAvailability is expected to make exactly one per call, batching every
// artist into a single round trip rather than looking each one up alone.
type fakeCache struct {
	entries          map[string]cacheEntry
	stored           map[string]cacheEntry
	lookupBatchCalls int
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
			{Artist: "Bring Me The Horizon", Available: true, JoysoundURL: "https://www.joysound.com/web/search/artist/62831"},
			{Artist: "Thornhill", Available: false},
			{Artist: "Architects", Available: true, JoysoundURL: "https://www.joysound.com/web/search/artist/1"},
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("batches every artist's cache lookup into a single round trip", func(t *testing.T) {
		cache := &fakeCache{entries: map[string]cacheEntry{
			"Architects":           {Available: true, LastChecked: time.Now()},
			"Bring Me The Horizon": {Available: false, LastChecked: time.Now().Add(-(cacheTTL + time.Hour))},
		}}
		svc := &Service{
			joysound: fakeCatalog{artists: map[string][]Artist{
				"Bring Me The Horizon": {{ID: "62831", Name: "Bring Me The Horizon"}},
			}},
			cache:   cache,
			limiter: noopLimiter{},
		}

		got, err := svc.CheckAvailability(context.Background(), []string{"Architects", "Bring Me The Horizon", "Thornhill"})
		if err != nil {
			t.Fatalf("CheckAvailability returned error: %v", err)
		}

		if cache.lookupBatchCalls != 1 {
			t.Errorf("got %d lookupBatch calls, want exactly 1 — every artist's cache check should be one round trip, not one per artist", cache.lookupBatchCalls)
		}

		want := []AvailabilityResult{
			{Artist: "Architects", Available: true},                                                                   // fresh cache hit — no ID in fake entry
			{Artist: "Bring Me The Horizon", Available: true, JoysoundURL: "https://www.joysound.com/web/search/artist/62831"}, // stale, refreshed live
			{Artist: "Thornhill", Available: false},                                                                   // cache miss, not found live
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("populates JoysoundURL from catalog artist ID", func(t *testing.T) {
		cache := &fakeCache{entries: map[string]cacheEntry{
			"Architects": {Available: true, CatalogArtistID: "62831", LastChecked: time.Now()},
			"Thornhill":  {Available: false, CatalogArtistID: "", LastChecked: time.Now()},
		}}
		svc := &Service{joysound: fakeCatalog{}, cache: cache, limiter: noopLimiter{}}

		got, err := svc.CheckAvailability(context.Background(), []string{"Architects", "Thornhill"})
		if err != nil {
			t.Fatalf("CheckAvailability returned error: %v", err)
		}

		if got[0].JoysoundURL != "https://www.joysound.com/web/search/artist/62831" {
			t.Errorf("Architects JoysoundURL = %q, want JOYSOUND artist link", got[0].JoysoundURL)
		}
		if got[1].JoysoundURL != "" {
			t.Errorf("Thornhill JoysoundURL = %q, want empty (not found on JOYSOUND)", got[1].JoysoundURL)
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
