package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/andrewdeanphillips/karaoke-match/backend/internal/spotify"
)

const stateCookieName = "spotify_auth_state"

// sessionCookieName names the long-lived, HttpOnly cookie that carries a
// visitor's session ID — the same opaque value stored as spotify_sessions.id,
// so resolving it is a single lookup by primary key. 30 days balances not
// asking returning visitors to log in again against Spotify's own refresh
// token lifetime.
const sessionCookieName = "spotify_session"

const sessionCookieMaxAge = 30 * 24 * 60 * 60 // 30 days, in seconds

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

	sessionID, err := a.spotify.ExchangeCode(r.Context(), code)
	if err != nil {
		log.Printf("spotify callback: exchanging code failed: %v", err)
		http.Error(w, "failed to complete Spotify authorization", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   sessionCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Spotify authorization complete — you can close this tab.")
}

// requireSpotifySession wraps a handler so it only runs for visitors with a
// valid Spotify session: it resolves the session cookie to a fresh access
// token — refreshing it first if it's gone stale — and carries that token in
// the request context for the handler to use (see spotify.AccessTokenFromContext).
// Visitors without a usable session are turned away with 401 before the
// wrapped handler ever runs.
func (a *api) requireSpotifySession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Error(w, "log in with Spotify first", http.StatusUnauthorized)
			return
		}

		token, err := a.spotify.AccessToken(r.Context(), cookie.Value)
		if err != nil {
			http.Error(w, "log in with Spotify first", http.StatusUnauthorized)
			return
		}

		next(w, r.WithContext(spotify.ContextWithAccessToken(r.Context(), token)))
	}
}
