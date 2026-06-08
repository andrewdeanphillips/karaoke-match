package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExampleRateLimiterPerIPCooldown(t *testing.T) {
	limiter := newExampleRateLimiter(20*time.Millisecond, 100)

	if !limiter.allow("1.2.3.4") {
		t.Fatal("expected the first request from an address to be allowed")
	}
	if limiter.allow("1.2.3.4") {
		t.Fatal("expected an immediate second request from the same address to be blocked")
	}
	if !limiter.allow("5.6.7.8") {
		t.Fatal("expected a request from a different address to be allowed regardless of the first address's cooldown")
	}

	time.Sleep(25 * time.Millisecond)
	if !limiter.allow("1.2.3.4") {
		t.Fatal("expected a request to be allowed again once its cooldown has elapsed")
	}
}

func TestExampleRateLimiterHourlyCap(t *testing.T) {
	limiter := newExampleRateLimiter(0, 2)

	if !limiter.allow("1.1.1.1") {
		t.Fatal("expected the first request to be allowed")
	}
	if !limiter.allow("2.2.2.2") {
		t.Fatal("expected the second request to be allowed")
	}
	if limiter.allow("3.3.3.3") {
		t.Fatal("expected a third request within the hour to be blocked by the hourly cap, regardless of address")
	}
}

func TestExampleRateLimiterHourlyCapResets(t *testing.T) {
	limiter := newExampleRateLimiter(0, 1)
	limiter.hourStart = time.Now().Add(-2 * time.Hour)
	limiter.hourCount = 1

	if !limiter.allow("9.9.9.9") {
		t.Fatal("expected the cap to reset once an hour has elapsed since the window started")
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		forwarded  string
		remoteAddr string
		want       string
	}{
		{
			name:       "prefers the first X-Forwarded-For address",
			forwarded:  "203.0.113.7, 10.0.0.1, 10.0.0.2",
			remoteAddr: "10.0.0.2:443",
			want:       "203.0.113.7",
		},
		{
			name:       "falls back to RemoteAddr when there's no forwarded header",
			remoteAddr: "198.51.100.23:54321",
			want:       "198.51.100.23",
		},
		{
			name:       "falls back to the raw RemoteAddr when it carries no port",
			remoteAddr: "198.51.100.23",
			want:       "198.51.100.23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/examples/match", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.forwarded != "" {
				r.Header.Set("X-Forwarded-For", tt.forwarded)
			}

			if got := clientIP(r); got != tt.want {
				t.Errorf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
