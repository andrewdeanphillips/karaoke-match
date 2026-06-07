package playlist

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
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

	artists, err := h.service.Import(r.Context(), req.URL)
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
