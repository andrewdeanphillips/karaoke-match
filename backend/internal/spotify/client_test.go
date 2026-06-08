package spotify

import (
	"context"
	"net/url"
	"testing"
)

func TestAuthURL(t *testing.T) {
	client := NewClient("test-client-id", "test-client-secret", "http://127.0.0.1:8080/callback")

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

func TestGetUserTokenWithoutSession(t *testing.T) {
	client := NewClient("test-client-id", "test-client-secret", "http://127.0.0.1:8080/callback")

	if _, err := client.GetUserToken(context.Background()); err == nil {
		t.Fatal("expected an error when no Spotify user session has been established yet")
	}
}
