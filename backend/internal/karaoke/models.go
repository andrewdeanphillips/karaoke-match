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

// AvailabilityResult pairs an artist name with whether JOYSOUND lists them —
// the unit of output for a playlist-wide availability check. JoysoundURL is
// the direct link to the artist's JOYSOUND page; it is only present when the
// artist was found (available artists always have one, unavailable ones never do).
type AvailabilityResult struct {
	Artist     string `json:"artist"`
	Available  bool   `json:"available"`
	JoysoundURL string `json:"joysoundUrl,omitempty"`
}
