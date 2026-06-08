package spotify

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const tokenURL = "https://accounts.spotify.com/api/token"
const authorizeURL = "https://accounts.spotify.com/authorize"
const apiBaseURL = "https://api.spotify.com/v1"

// expiryBuffer causes us to treat a token as stale slightly before Spotify
// actually expires it, so an in-flight request never gets caught out by expiry.
const expiryBuffer = 60 * time.Second

// authScope is the only Spotify scope required to read a playlist's tracks —
// per Spotify's API spec, this is mandatory even for public playlists.
const authScope = "playlist-read-private"

type Client struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	redirectURI  string

	sessions sessionStore
}

func NewClient(clientID, clientSecret, redirectURI string, pool *pgxpool.Pool) *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		sessions:     newPostgresSessionStore(pool),
	}
}

// AuthURL returns the Spotify authorization page URL to redirect a user's
// browser to, beginning the Authorization Code flow. state is an
// unpredictable value the caller generates (see GenerateState) and must
// verify on the callback to guard against CSRF attacks.
func (c *Client) AuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", c.redirectURI)
	params.Set("scope", authScope)
	params.Set("state", state)
	return authorizeURL + "?" + params.Encode()
}

// GenerateState returns a random, URL-safe string suitable for the OAuth
// "state" parameter — an unguessable value that ties an authorization
// request to its callback, preventing cross-site request forgery.
func GenerateState() (string, error) {
	return randomToken()
}

// randomToken returns a random, URL-safe string suitable for any value that
// must be unguessable: OAuth state, session IDs, and the like.
func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ExchangeCode trades an authorization code (received on the OAuth callback)
// for a user access token and refresh token, persists them as a new session,
// and returns that session's ID — the value the caller should hand back to
// the visitor's browser as a cookie.
func (c *Client) ExchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.redirectURI)

	token, err := c.requestUserToken(ctx, form)
	if err != nil {
		return "", fmt.Errorf("exchanging authorization code: %w", err)
	}

	id, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("generating session ID: %w", err)
	}

	sess := session{
		ID:           id,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(token.ExpiresIn)*time.Second - expiryBuffer),
	}
	if err := c.sessions.create(ctx, sess); err != nil {
		return "", fmt.Errorf("creating session: %w", err)
	}

	return id, nil
}

// AccessToken returns a valid access token for the given session, transparently
// using the session's refresh token to obtain and persist a new one when the
// stored token has expired.
func (c *Client) AccessToken(ctx context.Context, sessionID string) (string, error) {
	sess, ok, err := c.sessions.lookup(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("looking up session: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("no Spotify session found — visit /auth/login first")
	}

	if time.Now().Before(sess.ExpiresAt) {
		return sess.AccessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", sess.RefreshToken)

	token, err := c.requestUserToken(ctx, form)
	if err != nil {
		return "", fmt.Errorf("refreshing user token: %w", err)
	}

	// Spotify doesn't always issue a new refresh token on renewal — keep the
	// existing one whenever it omits a replacement.
	refreshToken := sess.RefreshToken
	if token.RefreshToken != "" {
		refreshToken = token.RefreshToken
	}
	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn)*time.Second - expiryBuffer)

	if err := c.sessions.updateTokens(ctx, sessionID, token.AccessToken, refreshToken, expiresAt); err != nil {
		return "", fmt.Errorf("persisting refreshed session: %w", err)
	}

	return token.AccessToken, nil
}

// requestUserToken performs the token-endpoint exchange shared by the
// authorization-code and refresh-token grants — they differ only in which
// form fields they send, so the request/decode plumbing is factored out here.
func (c *Client) requestUserToken(ctx context.Context, form url.Values) (*userTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building user token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting user token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify user token request failed with status %d", resp.StatusCode)
	}

	var token userTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decoding user token response: %w", err)
	}

	return &token, nil
}

// GetPlaylistTracks returns every track in the given playlist, following
// Spotify's pagination until there are no more pages left. accessToken is the
// caller's responsibility to resolve (see AccessToken) — this method has no
// opinion about whose session it belongs to.
func (c *Client) GetPlaylistTracks(ctx context.Context, playlistID, accessToken string) ([]Track, error) {
	var tracks []Track
	pageURL := fmt.Sprintf("%s/playlists/%s/items", apiBaseURL, playlistID)

	for pageURL != "" {
		page, err := c.fetchPlaylistTracksPage(ctx, pageURL, accessToken)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			if item.Item.Type != "track" {
				continue
			}
			tracks = append(tracks, toTrack(item.Item))
		}
		pageURL = page.Next
	}

	return tracks, nil
}

func (c *Client) fetchPlaylistTracksPage(ctx context.Context, pageURL, token string) (*playlistTracksPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building playlist tracks request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting playlist tracks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify playlist tracks request failed with status %d", resp.StatusCode)
	}

	var page playlistTracksPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("decoding playlist tracks page: %w", err)
	}

	return &page, nil
}

func toTrack(t trackObject) Track {
	artists := make([]string, len(t.Artists))
	for i, a := range t.Artists {
		artists[i] = a.Name
	}
	return Track{Name: t.Name, Artists: artists}
}
