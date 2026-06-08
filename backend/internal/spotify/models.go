package spotify

// userTokenResponse mirrors the fields we use from Spotify's token response
// for the Authorization Code and refresh_token grants.
type userTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// playlistTracksPage mirrors the fields we use from one page of Spotify's
// playlist-tracks response: items and a link to the next page (empty when
// there is none).
type playlistTracksPage struct {
	Items []playlistItem `json:"items"`
	Next  string         `json:"next"`
}

type playlistItem struct {
	Item trackObject `json:"item"`
}

// trackObject mirrors a playlist entry's content. Spotify playlists can hold
// either tracks or podcast episodes — Type tells us which, so we can skip
// episodes (they carry no useful Artists data for our matching purposes).
type trackObject struct {
	Type    string         `json:"type"`
	Name    string         `json:"name"`
	Artists []artistObject `json:"artists"`
}

type artistObject struct {
	Name string `json:"name"`
}

// Track is the playlist data we expose beyond this package — just enough to
// extract artists and match against a karaoke catalog, without leaking
// Spotify's JSON shape.
type Track struct {
	Name    string
	Artists []string
}
