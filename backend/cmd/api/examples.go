package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// exampleIPCooldown is the minimum spacing exampleRateLimiter enforces
// between requests from the same visitor — generous enough to click through
// all of curatedExamples at a natural pace, tight enough that scripting the
// endpoint gains an attacker almost nothing.
const exampleIPCooldown = 15 * time.Second

// exampleHourlyCap bounds total /examples/match calls across all visitors in
// a rolling hour. It's the backstop against the per-IP cooldown: a visitor
// spread across many addresses could still slip past that, but never past
// this — so the one shared session's Spotify usage stays bounded no matter
// how an abuser distributes their requests.
const exampleHourlyCap = 100

// exampleRateLimiter protects /examples/match — the one public, no-login
// endpoint that spends Andrew's personal Spotify session on an anonymous
// visitor's behalf. Two checks share a single lock: a per-IP cooldown guards
// against any one visitor hammering it, and a rolling hourly cap guards
// against the same abuse spread across many addresses.
type exampleRateLimiter struct {
	cooldown  time.Duration
	hourlyCap int

	mu        sync.Mutex
	lastSeen  map[string]time.Time
	hourStart time.Time
	hourCount int
}

func newExampleRateLimiter(cooldown time.Duration, hourlyCap int) *exampleRateLimiter {
	return &exampleRateLimiter{
		cooldown:  cooldown,
		hourlyCap: hourlyCap,
		lastSeen:  make(map[string]time.Time),
		hourStart: time.Now(),
	}
}

// allow reports whether a request from ip should proceed, consuming its
// share of both budgets if so.
func (l *exampleRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.hourStart) >= time.Hour {
		l.hourStart = now
		l.hourCount = 0
	}

	if l.hourCount >= l.hourlyCap {
		return false
	}
	if last, seen := l.lastSeen[ip]; seen && now.Sub(last) < l.cooldown {
		return false
	}

	l.lastSeen[ip] = now
	l.hourCount++
	return true
}

// clientIP returns the address a rate limiter should key on. Cloud Run
// terminates every connection at its own load balancer and forwards the
// original visitor's address as the first entry of X-Forwarded-For — a
// header a visitor has no way to set themselves, since their request never
// reaches this server directly. Run locally, with nothing in front of it,
// RemoteAddr already is the visitor's address.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if ip := strings.TrimSpace(strings.Split(forwarded, ",")[0]); ip != "" {
			return ip
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// curatedExample is one playlist hand-picked for the "try an example" path —
// chosen by Andrew so a visitor with no Spotify account of their own (a
// recruiter, say) can still watch KaraokeMatch run end-to-end with one click.
// ID is an opaque slug the frontend passes back to identify which one to run;
// keeping it separate from the Spotify playlist ID means the client never
// needs to know — or be trusted to send — a real playlist URL for this path.
type curatedExample struct {
	ID   string
	Name string
	URL  string
}

var curatedExamples = []curatedExample{
	{ID: "japanese-heavy", Name: "Japanese Heavy", URL: "https://open.spotify.com/playlist/5HUzT1zcW1XFdVoI7mrtyV"},
	{ID: "mandarin-playlist", Name: "mandarin playlist 🍊", URL: "https://open.spotify.com/playlist/5PbnP52xzTi3vHTGKnyXZo"},
	{ID: "loading", Name: "Loading...", URL: "https://open.spotify.com/playlist/0s2afFzM9nLeDhQDOEhO9V"},
}

func findCuratedExample(id string) (curatedExample, bool) {
	for _, example := range curatedExamples {
		if example.ID == id {
			return example, true
		}
	}
	return curatedExample{}, false
}

type exampleSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type examplesResponse struct {
	Examples []exampleSummary `json:"examples"`
}

// examplesListHandler handles GET /examples: it lists the curated playlists
// behind the "try an example" path. When exampleSessionID isn't configured —
// the local default — it reports an empty list, so the frontend knows to hide
// the feature entirely rather than offer links that can only ever fail.
func (a *api) examplesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	examples := []exampleSummary{}
	if a.exampleSessionID != "" {
		for _, example := range curatedExamples {
			examples = append(examples, exampleSummary{ID: example.ID, Name: example.Name})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(examplesResponse{Examples: examples})
}

type exampleMatchRequest struct {
	ID string `json:"id"`
}

// exampleMatchHandler handles POST /examples/match: it runs the same
// import-and-check flow as /playlist/match, but against one of the curated
// playlists above and authenticated with the one deliberate, owner-held
// session named by exampleSessionID — so visitors without a Spotify account
// of their own can still see the app work end-to-end, no login required.
func (a *api) exampleMatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if a.exampleSessionID == "" {
		http.Error(w, "the example path is not available", http.StatusNotFound)
		return
	}

	if !a.exampleLimiter.allow(clientIP(r)) {
		http.Error(w, "too many example requests — please try again in a moment", http.StatusTooManyRequests)
		return
	}

	var req exampleMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	example, ok := findCuratedExample(req.ID)
	if !ok {
		http.Error(w, "unknown example playlist", http.StatusBadRequest)
		return
	}

	accessToken, err := a.spotify.AccessToken(r.Context(), a.exampleSessionID)
	if err != nil {
		log.Printf("example match: resolving owner session: %v", err)
		http.Error(w, "failed to load the example playlist", http.StatusInternalServerError)
		return
	}

	a.runMatch(r.Context(), w, example.URL, accessToken)
}
