package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/andrewdeanphillips/karaoke-match/backend/internal/karaoke"
	"github.com/andrewdeanphillips/karaoke-match/backend/internal/playlist"
	"github.com/andrewdeanphillips/karaoke-match/backend/internal/spotify"
)

// matchResponse is the JSON body returned by the playlist match endpoint.
// Results may be shorter than TotalArtists — CheckAvailability caps how many
// live JOYSOUND searches a single check will make, so a playlist with many
// uncached artists can end up only partially checked. TotalArtists lets
// callers distinguish that from a complete check rather than mistaking a
// short list for the full picture.
type matchResponse struct {
	Results      []karaoke.AvailabilityResult `json:"results"`
	TotalArtists int                          `json:"totalArtists"`
}

// matchHandler handles POST /playlist/match: it accepts a Spotify playlist
// URL, imports the artists credited on it, checks each one's availability on
// JOYSOUND, and responds with the combined per-artist match summary.
func (a *api) matchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req playlist.ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	accessToken, ok := spotify.AccessTokenFromContext(r.Context())
	if !ok {
		log.Print("playlist match: no access token in context — is the session middleware wired up?")
		http.Error(w, "failed to import playlist", http.StatusInternalServerError)
		return
	}

	a.runMatch(r.Context(), w, req.URL, accessToken)
}

// runMatch imports the artists credited on a playlist — authenticating with
// Spotify via the given access token — checks each one's availability on
// JOYSOUND, and writes the combined match response. It's the shared core of
// matchHandler and exampleMatchHandler: both end up running exactly this
// sequence, differing only in whose access token they hand it and how they
// got hold of one.
func (a *api) runMatch(ctx context.Context, w http.ResponseWriter, playlistURL, accessToken string) {
	artists, err := a.playlist.Import(ctx, playlistURL, accessToken)
	if err != nil {
		if errors.Is(err, playlist.ErrInvalidPlaylistURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("playlist match: importing playlist: %v", err)
		http.Error(w, "failed to import playlist", http.StatusInternalServerError)
		return
	}

	results, err := a.karaoke.CheckAvailability(ctx, artists)
	if err != nil {
		log.Printf("playlist match: checking availability: %v", err)
		http.Error(w, "failed to check artist availability", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchResponse{Results: results, TotalArtists: len(artists)})
}
