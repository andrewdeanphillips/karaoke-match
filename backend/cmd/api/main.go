package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/andrewdeanphillips/karaoke-match/backend/internal/database"
	"github.com/andrewdeanphillips/karaoke-match/backend/internal/karaoke"
	"github.com/andrewdeanphillips/karaoke-match/backend/internal/playlist"
	"github.com/andrewdeanphillips/karaoke-match/backend/internal/spotify"
)

type api struct {
	db             *pgxpool.Pool
	spotify        *spotify.Client
	karaoke        *karaoke.Service
	playlist       *playlist.Service
	frontendOrigin string
}

func withCORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		// Visitor sessions ride on a cookie, so the browser must be told it's
		// allowed to send and store credentials on cross-origin requests —
		// it otherwise strips Set-Cookie from responses and Cookie from requests.
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *api) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status := "ok"
	if err := a.db.Ping(ctx); err != nil {
		status = "degraded"
		log.Printf("health check: database ping failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   status,
		"database": status,
	})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on existing environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	frontendOrigin := os.Getenv("FRONTEND_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:5173"
	}

	spotifyClientID := os.Getenv("SPOTIFY_CLIENT_ID")
	spotifyClientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	spotifyRedirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")
	if spotifyClientID == "" || spotifyClientSecret == "" || spotifyRedirectURI == "" {
		log.Fatal("SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET, and SPOTIFY_REDIRECT_URI environment variables are required")
	}
	ctx := context.Background()
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	spotifyClient := spotify.NewClient(spotifyClientID, spotifyClientSecret, spotifyRedirectURI, pool)

	playlistService := playlist.NewService(spotifyClient)
	a := &api{db: pool, spotify: spotifyClient, karaoke: karaoke.NewService(pool), playlist: playlistService, frontendOrigin: frontendOrigin}
	playlistHandler := playlist.NewHandler(playlistService)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.healthHandler)
	mux.HandleFunc("/auth/login", a.spotifyLoginHandler)
	mux.HandleFunc("/auth/session", a.requireSpotifySession(a.spotifySessionHandler))
	mux.HandleFunc("/callback", a.spotifyCallbackHandler)
	mux.HandleFunc("/playlist/import", a.requireSpotifySession(playlistHandler.Import))
	mux.HandleFunc("/playlist/match", a.requireSpotifySession(a.matchHandler))

	handler := withCORS(frontendOrigin, mux)

	addr := ":" + port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
