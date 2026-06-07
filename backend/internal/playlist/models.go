package playlist

// ImportRequest is the JSON body accepted by the playlist import endpoint.
type ImportRequest struct {
	URL string `json:"url"`
}

// ImportResponse is the JSON body returned by the playlist import endpoint.
type ImportResponse struct {
	Artists []string `json:"artists"`
}
