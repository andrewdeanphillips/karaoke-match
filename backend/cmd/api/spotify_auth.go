package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/andrewdeanphillips/karaoke-match/backend/internal/spotify"
)

const stateCookieName = "spotify_auth_state"

// spotifyLoginHandler starts the Authorization Code flow: it generates a
// CSRF-guarding state value, remembers it in a short-lived cookie, and
// redirects the browser to Spotify's consent page.
func (a *api) spotifyLoginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := spotify.GenerateState()
	if err != nil {
		log.Printf("spotify login: generating state failed: %v", err)
		http.Error(w, "failed to start Spotify authorization", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, a.spotify.AuthURL(state), http.StatusFound)
}

// spotifyCallbackHandler completes the flow: Spotify redirects the browser
// back here with an authorization code and the state we issued. We confirm
// the state matches our cookie (guarding against CSRF), then exchange the
// code for a user access token and refresh token.
func (a *api) spotifyCallbackHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(stateCookieName)
	if err != nil || r.URL.Query().Get("state") != cookie.Value {
		http.Error(w, "invalid or missing OAuth state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   stateCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	if err := a.spotify.ExchangeCode(r.Context(), code); err != nil {
		log.Printf("spotify callback: exchanging code failed: %v", err)
		http.Error(w, "failed to complete Spotify authorization", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Spotify authorization complete — you can close this tab.")
}
