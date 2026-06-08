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

// AvailabilityResult pairs an artist name with whether JOYSOUND lists them —
// the unit of output for a playlist-wide availability check.
type AvailabilityResult struct {
	Artist    string `json:"artist"`
	Available bool   `json:"available"`
}
