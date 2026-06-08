package karaoke

import "testing"

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
