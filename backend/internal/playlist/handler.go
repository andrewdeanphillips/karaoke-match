package playlist

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/andrewdeanphillips/karaoke-match/backend/internal/spotify"
)

// Handler exposes the playlist domain over HTTP.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Import handles POST /playlist/import: it accepts a Spotify playlist URL and
// responds with the unique artists credited across that playlist's tracks.
// It must run behind the Spotify session middleware, which resolves the
// visitor's access token into the request context before this ever runs.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	accessToken, ok := spotify.AccessTokenFromContext(r.Context())
	if !ok {
		log.Print("playlist import: no access token in context — is the session middleware wired up?")
		http.Error(w, "failed to import playlist", http.StatusInternalServerError)
		return
	}

	artists, err := h.service.Import(r.Context(), req.URL, accessToken)
	if err != nil {
		if errors.Is(err, ErrInvalidPlaylistURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("playlist import: %v", err)
		http.Error(w, "failed to import playlist", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ImportResponse{Artists: artists})
}
