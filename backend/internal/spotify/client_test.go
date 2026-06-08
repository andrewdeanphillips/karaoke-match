package spotify

import (
	"context"
	"net/url"
	"testing"
	"time"
)

// fakeSessionStore substitutes for postgresSessionStore in tests, letting us
// exercise Client's session lookup and refresh logic without a real database.
type fakeSessionStore struct {
	sessions map[string]session
	updated  map[string]session
}

func (f *fakeSessionStore) create(_ context.Context, sess session) error {
	if f.sessions == nil {
		f.sessions = make(map[string]session)
	}
	f.sessions[sess.ID] = sess
	return nil
}

func (f *fakeSessionStore) lookup(_ context.Context, id string) (session, bool, error) {
	sess, found := f.sessions[id]
	return sess, found, nil
}

func (f *fakeSessionStore) updateTokens(_ context.Context, id, accessToken, refreshToken string, expiresAt time.Time) error {
	if f.updated == nil {
		f.updated = make(map[string]session)
	}
	f.updated[id] = session{ID: id, AccessToken: accessToken, RefreshToken: refreshToken, ExpiresAt: expiresAt}
	return nil
}

func TestAuthURL(t *testing.T) {
	client := NewClient("test-client-id", "test-client-secret", "http://127.0.0.1:8080/callback", nil)

	parsed, err := url.Parse(client.AuthURL("test-state"))
	if err != nil {
		t.Fatalf("AuthURL produced an unparseable URL: %v", err)
	}

	query := parsed.Query()
	want := map[string]string{
		"client_id":     "test-client-id",
		"response_type": "code",
		"redirect_uri":  "http://127.0.0.1:8080/callback",
		"scope":         authScope,
		"state":         "test-state",
	}
	for key, expected := range want {
		if got := query.Get(key); got != expected {
			t.Errorf("query parameter %q = %q, want %q", key, got, expected)
		}
	}
}

func TestGenerateState(t *testing.T) {
	first, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState returned error: %v", err)
	}
	second, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState returned error: %v", err)
	}

	if first == "" {
		t.Fatal("expected a non-empty state value")
	}
	if first == second {
		t.Fatal("expected two calls to produce different state values")
	}
}

func TestAccessTokenWithoutSession(t *testing.T) {
	client := &Client{sessions: &fakeSessionStore{}}

	if _, err := client.AccessToken(context.Background(), "unknown-session-id"); err == nil {
		t.Fatal("expected an error when no session exists for the given ID")
	}
}

func TestAccessTokenReturnsCachedTokenWhenStillValid(t *testing.T) {
	store := &fakeSessionStore{sessions: map[string]session{
		"session-1": {
			ID:           "session-1",
			AccessToken:  "cached-access-token",
			RefreshToken: "cached-refresh-token",
			ExpiresAt:    time.Now().Add(time.Hour),
		},
	}}
	client := &Client{sessions: store}

	token, err := client.AccessToken(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("AccessToken returned error: %v", err)
	}
	if token != "cached-access-token" {
		t.Errorf("token = %q, want the cached access token", token)
	}
	if store.updated != nil {
		t.Error("expected no refresh to occur for a still-valid token")
	}
}
