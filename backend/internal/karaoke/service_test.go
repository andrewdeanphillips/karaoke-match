package karaoke

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
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

func TestAnyArtistNamed(t *testing.T) {
	tests := []struct {
		name    string
		artists []Artist
		query   string
		want    bool
	}{
		{
			name:    "no results",
			artists: nil,
			query:   "Bring Me The Horizon",
			want:    false,
		},
		{
			name: "exact match",
			artists: []Artist{
				{ID: "62831", Name: "Bring Me The Horizon"},
			},
			query: "Bring Me The Horizon",
			want:  true,
		},
		{
			name: "case-insensitive match",
			artists: []Artist{
				{ID: "62831", Name: "bring me the horizon"},
			},
			query: "Bring Me The Horizon",
			want:  true,
		},
		{
			name: "different artist of the same general search is not a match",
			artists: []Artist{
				{ID: "423215", Name: "MACHINE GUN KELLY"},
			},
			query: "Bring Me The Horizon",
			want:  false,
		},
		{
			name: "match present among unrelated results",
			artists: []Artist{
				{ID: "10585", Name: "Def Tech"},
				{ID: "62831", Name: "Bring Me The Horizon"},
			},
			query: "Bring Me The Horizon",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := anyArtistNamed(tt.artists, tt.query); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoundArtists(t *testing.T) {
	artists := make([]string, maxArtistsPerCheck+10)
	for i := range artists {
		artists[i] = fmt.Sprintf("Artist %d", i)
	}

	tests := []struct {
		name    string
		artists []string
		want    int
	}{
		{name: "fewer than the cap", artists: artists[:maxArtistsPerCheck-1], want: maxArtistsPerCheck - 1},
		{name: "exactly the cap", artists: artists[:maxArtistsPerCheck], want: maxArtistsPerCheck},
		{name: "more than the cap", artists: artists, want: maxArtistsPerCheck},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boundArtists(tt.artists)
			if len(got) != tt.want {
				t.Fatalf("got %d artists, want %d", len(got), tt.want)
			}
			for i, artist := range got {
				if artist != tt.artists[i] {
					t.Errorf("artist %d: got %q, want %q", i, artist, tt.artists[i])
				}
			}
		})
	}
}

func TestCheckAvailability(t *testing.T) {
	t.Run("aggregates a result per artist, in order", func(t *testing.T) {
		svc := &Service{joysound: fakeCatalog{artists: map[string][]Artist{
			"Bring Me The Horizon": {{ID: "62831", Name: "Bring Me The Horizon"}},
			"Architects":           {{ID: "1", Name: "Architects"}},
		}}}

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
		svc := &Service{joysound: fakeCatalog{err: errors.New("boom")}}

		if _, err := svc.CheckAvailability(context.Background(), []string{"Architects"}); err == nil {
			t.Error("expected an error, got nil")
		}
	})
}
