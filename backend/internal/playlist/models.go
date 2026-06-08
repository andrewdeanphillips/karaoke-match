package playlist

// ImportRequest is the JSON body accepted by the playlist match endpoint.
type ImportRequest struct {
	URL string `json:"url"`
}
