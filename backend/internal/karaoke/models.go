package karaoke

// Artist is a karaoke-catalog search result that JOYSOUND itself classifies
// as an artist match — distinct from song results, which the search page
// renders in a separate section. ID is JOYSOUND's own stable identifier
// (e.g. the "62831" in /web/search/artist/62831), kept alongside the display
// name for future use (caching, cross-referencing) even though matching
// today is name-based.
type Artist struct {
	ID   string
	Name string
}

// Song is a karaoke-catalog search result that JOYSOUND itself classifies as
// a song match — distinct from artist results, which the search page renders
// in a separate section. ID is JOYSOUND's own stable identifier (e.g. the
// "82077" in /web/search/song/82077). Artist is the song's credited artist as
// JOYSOUND lists it, used to confirm a title match is for the right artist.
type Song struct {
	ID     string
	Title  string
	Artist string
}

// Track is a playlist track to check for song-level availability — just
// enough to search and match against JOYSOUND, independent of where it
// came from.
type Track struct {
	Artist string
	Title  string
}

// TrackAvailabilityResult pairs a playlist track with links to its match (or
// matches) on JOYSOUND. ArtistJoysoundURL and SongJoysoundURL are independent
// — either, both, or neither may be present, depending on whether JOYSOUND
// has the track's artist, the specific song, or both.
type TrackAvailabilityResult struct {
	Artist            string `json:"artist"`
	Title             string `json:"title"`
	ArtistJoysoundURL string `json:"artistJoysoundUrl,omitempty"`
	SongJoysoundURL   string `json:"songJoysoundUrl,omitempty"`
}
