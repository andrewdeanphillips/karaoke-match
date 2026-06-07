package spotify

import (
	"context"
	"os"
	"testing"
)

func TestGetToken(t *testing.T) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		t.Skip("SPOTIFY_CLIENT_ID/SPOTIFY_CLIENT_SECRET not set; skipping live Spotify check")
	}

	client := NewClient(clientID, clientSecret)
	ctx := context.Background()

	first, err := client.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken returned error: %v", err)
	}
	if first == "" {
		t.Fatal("expected a non-empty access token")
	}

	second, err := client.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken returned error on second call: %v", err)
	}
	if second != first {
		t.Fatal("expected second call to return the cached token, got a different one")
	}

	t.Logf("received token (truncated: %s...), cache returned same token on second call", first[:8])
}
