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
	"sync"
	"time"
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

	mu        sync.Mutex
	token     string
	expiresAt time.Time

	userMu       sync.Mutex
	userToken    string
	userExpires  time.Time
	refreshToken string
}

func NewClient(clientID, clientSecret, redirectURI string) *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
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
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetToken returns a cached access token if one is still valid, fetching a
// fresh one from Spotify when there is none or it has expired.
func (c *Client) GetToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.expiresAt) {
		return c.token, nil
	}

	token, expiresIn, err := c.fetchAccessToken(ctx)
	if err != nil {
		return "", err
	}

	c.token = token
	c.expiresAt = time.Now().Add(expiresIn - expiryBuffer)

	return c.token, nil
}

func (c *Client) fetchAccessToken(ctx context.Context) (string, time.Duration, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("requesting access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("spotify token request failed with status %d", resp.StatusCode)
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", 0, fmt.Errorf("decoding token response: %w", err)
	}

	return token.AccessToken, time.Duration(token.ExpiresIn) * time.Second, nil
}

// ExchangeCode trades an authorization code (received on the OAuth callback)
// for a user access token and refresh token, caching both for GetUserToken.
func (c *Client) ExchangeCode(ctx context.Context, code string) error {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.redirectURI)

	token, err := c.requestUserToken(ctx, form)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}

	c.userMu.Lock()
	defer c.userMu.Unlock()
	c.userToken = token.AccessToken
	c.userExpires = time.Now().Add(time.Duration(token.ExpiresIn)*time.Second - expiryBuffer)
	c.refreshToken = token.RefreshToken

	return nil
}

// GetUserToken returns the cached user access token, transparently using the
// refresh token to obtain a new one when the cached token has expired.
func (c *Client) GetUserToken(ctx context.Context) (string, error) {
	c.userMu.Lock()
	defer c.userMu.Unlock()

	if c.userToken != "" && time.Now().Before(c.userExpires) {
		return c.userToken, nil
	}

	if c.refreshToken == "" {
		return "", fmt.Errorf("no Spotify user session — visit /auth/login first")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", c.refreshToken)

	token, err := c.requestUserToken(ctx, form)
	if err != nil {
		return "", fmt.Errorf("refreshing user token: %w", err)
	}

	c.userToken = token.AccessToken
	c.userExpires = time.Now().Add(time.Duration(token.ExpiresIn)*time.Second - expiryBuffer)
	// Spotify doesn't always issue a new refresh token on renewal — keep the
	// existing one whenever it omits a replacement.
	if token.RefreshToken != "" {
		c.refreshToken = token.RefreshToken
	}

	return c.userToken, nil
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
// Spotify's pagination until there are no more pages left.
func (c *Client) GetPlaylistTracks(ctx context.Context, playlistID string) ([]Track, error) {
	token, err := c.GetUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting user access token: %w", err)
	}

	var tracks []Track
	pageURL := fmt.Sprintf("%s/playlists/%s/items", apiBaseURL, playlistID)

	for pageURL != "" {
		page, err := c.fetchPlaylistTracksPage(ctx, pageURL, token)
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
