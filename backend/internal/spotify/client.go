package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const tokenURL = "https://accounts.spotify.com/api/token"

// expiryBuffer causes us to treat a token as stale slightly before Spotify
// actually expires it, so an in-flight request never gets caught out by expiry.
const expiryBuffer = 60 * time.Second

type Client struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
	}
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
