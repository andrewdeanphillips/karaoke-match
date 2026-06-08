package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// curatedExample is one playlist hand-picked for the "try an example" path —
// chosen by Andrew so a visitor with no Spotify account of their own (a
// recruiter, say) can still watch KaraokeMatch run end-to-end with one click.
// ID is an opaque slug the frontend passes back to identify which one to run;
// keeping it separate from the Spotify playlist ID means the client never
// needs to know — or be trusted to send — a real playlist URL for this path.
type curatedExample struct {
	ID   string
	Name string
	URL  string
}

var curatedExamples = []curatedExample{
	{ID: "japanese-heavy", Name: "Japanese Heavy", URL: "https://open.spotify.com/playlist/5HUzT1zcW1XFdVoI7mrtyV"},
	{ID: "mandarin-playlist", Name: "mandarin playlist 🍊", URL: "https://open.spotify.com/playlist/5PbnP52xzTi3vHTGKnyXZo"},
	{ID: "loading", Name: "Loading...", URL: "https://open.spotify.com/playlist/0s2afFzM9nLeDhQDOEhO9V"},
}

func findCuratedExample(id string) (curatedExample, bool) {
	for _, example := range curatedExamples {
		if example.ID == id {
			return example, true
		}
	}
	return curatedExample{}, false
}

type exampleSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type examplesResponse struct {
	Examples []exampleSummary `json:"examples"`
}

// examplesListHandler handles GET /examples: it lists the curated playlists
// behind the "try an example" path. When exampleSessionID isn't configured —
// the local default — it reports an empty list, so the frontend knows to hide
// the feature entirely rather than offer links that can only ever fail.
func (a *api) examplesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	examples := []exampleSummary{}
	if a.exampleSessionID != "" {
		for _, example := range curatedExamples {
			examples = append(examples, exampleSummary{ID: example.ID, Name: example.Name})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(examplesResponse{Examples: examples})
}

type exampleMatchRequest struct {
	ID string `json:"id"`
}

// exampleMatchHandler handles POST /examples/match: it runs the same
// import-and-check flow as /playlist/match, but against one of the curated
// playlists above and authenticated with the one deliberate, owner-held
// session named by exampleSessionID — so visitors without a Spotify account
// of their own can still see the app work end-to-end, no login required.
func (a *api) exampleMatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if a.exampleSessionID == "" {
		http.Error(w, "the example path is not available", http.StatusNotFound)
		return
	}

	var req exampleMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	example, ok := findCuratedExample(req.ID)
	if !ok {
		http.Error(w, "unknown example playlist", http.StatusBadRequest)
		return
	}

	accessToken, err := a.spotify.AccessToken(r.Context(), a.exampleSessionID)
	if err != nil {
		log.Printf("example match: resolving owner session: %v", err)
		http.Error(w, "failed to load the example playlist", http.StatusInternalServerError)
		return
	}

	a.runMatch(r.Context(), w, example.URL, accessToken)
}
