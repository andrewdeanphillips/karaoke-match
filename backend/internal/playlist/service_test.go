package playlist

import (
	"slices"
	"testing"

	"github.com/andrewdeanphillips/karaoke-match/backend/internal/spotify"
)

func TestParsePlaylistID(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantID  string
		wantErr bool
	}{
		{
			name:   "valid URL with query string",
			url:    "https://open.spotify.com/playlist/0s2afFzM9nLeDhQDOEhO9V?si=5ffa0a2abfa34ef4",
			wantID: "0s2afFzM9nLeDhQDOEhO9V",
		},
		{
			name:   "valid URL without query string",
			url:    "https://open.spotify.com/playlist/37i9dQZF1DX5Ejj0EkURtP",
			wantID: "37i9dQZF1DX5Ejj0EkURtP",
		},
		{
			name:    "wrong host",
			url:     "https://example.com/playlist/0s2afFzM9nLeDhQDOEhO9V",
			wantErr: true,
		},
		{
			name:    "wrong path",
			url:     "https://open.spotify.com/album/0s2afFzM9nLeDhQDOEhO9V",
			wantErr: true,
		},
		{
			name:    "missing ID",
			url:     "https://open.spotify.com/playlist/",
			wantErr: true,
		},
		{
			name:    "extra path segment",
			url:     "https://open.spotify.com/playlist/abc/extra",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := parsePlaylistID(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got ID %q", id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("got ID %q, want %q", id, tt.wantID)
			}
		})
	}
}

func TestUniqueArtists(t *testing.T) {
	tests := []struct {
		name   string
		tracks []spotify.Track
		want   []string
	}{
		{
			name:   "no tracks",
			tracks: nil,
			want:   nil,
		},
		{
			name: "deduplicates repeated artists, preserving first-seen order",
			tracks: []spotify.Track{
				{Name: "Track A", Artists: []string{"Artist X", "Artist Y"}},
				{Name: "Track B", Artists: []string{"Artist Y", "Artist Z"}},
				{Name: "Track C", Artists: []string{"Artist X"}},
			},
			want: []string{"Artist X", "Artist Y", "Artist Z"},
		},
		{
			name: "tracks with no artists contribute nothing",
			tracks: []spotify.Track{
				{Name: "Track A", Artists: nil},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueArtists(tt.tracks)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
