package karaoke

import (
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// artistResultHTML and songResultHTML are trimmed-down fixtures of JOYSOUND's
// actual cross-search result markup, keeping only the structure findArtists
// and findSongs depend on.
const artistResultHTML = `
<a href="/web/search/artist/62831">Bring Me The Horizon</a>
`

const songResultHTML = `
<a href="/web/search/song/82077">
  <div>
    <div>
      <ul></ul>
      <div class="flex flex-row gap-2">
        <p class="font-bold text-lg">Off The Edge feat.WISE</p>
      </div>
      <div class="font-medium text-sm">Def Tech</div>
    </div>
  </div>
</a>
`

func parseFragment(t *testing.T, fragment string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(fragment))
	if err != nil {
		t.Fatalf("parsing fragment: %v", err)
	}
	return doc
}

func TestFindArtists(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []Artist
	}{
		{
			name: "artist link",
			html: artistResultHTML,
			want: []Artist{{ID: "62831", Name: "Bring Me The Horizon"}},
		},
		{
			name: "song link is not an artist match",
			html: songResultHTML,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findArtists(parseFragment(t, tt.html))
			if !slices.Equal(got, tt.want) {
				t.Errorf("findArtists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindSongs(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []Song
	}{
		{
			name: "song link",
			html: songResultHTML,
			want: []Song{{ID: "82077", Title: "Off The Edge feat.WISE", Artist: "Def Tech"}},
		},
		{
			name: "artist link is not a song match",
			html: artistResultHTML,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSongs(parseFragment(t, tt.html))
			if !slices.Equal(got, tt.want) {
				t.Errorf("findSongs() = %v, want %v", got, tt.want)
			}
		})
	}
}
