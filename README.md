# KaraokeMatch

KaraokeMatch tells you which artists from a Spotify playlist are actually
available on [JOYSOUND](https://www.joysound.com/), a major Japanese karaoke
platform. Log in with Spotify, point it at one of your playlists, and get
back a per-artist breakdown of what JOYSOUND carries — built to solve a real
annoyance: manually cross-checking niche-genre playlists (metalcore,
post-hardcore, prog metal) against karaoke catalogs one search at a time.
No Spotify account? A "try an example" path on the landing page runs the
same flow against a few curated playlists with one click.

This is also a learning project — a hands-on way to build production-style
backend systems in Go, going deep on API design, database schema design,
caching, and automated testing rather than skimming the surface of a
tutorial.

## Project status

**MVP in progress — 7 of 8 milestones complete.** See
[`docs/MVP_ROADMAP.md`](docs/MVP_ROADMAP.md) for the full plan.

- [x] **Project setup** — Go backend, React/TypeScript frontend, PostgreSQL, CORS wiring between them
- [x] **Spotify playlist import** — OAuth Authorization Code flow, playlist parsing, unique-artist extraction
- [x] **JOYSOUND integration** — search-results scraping and artist matching, including handling same-name collisions across catalogs
- [x] **Playlist matching** — a single endpoint combining playlist import with per-artist availability checks
- [x] **Availability cache** — a Postgres-backed cache (with TTL) sitting in front of JOYSOUND, plus a rate limiter and a request budget that bounds how many live lookups one playlist match can trigger
- [x] **Results dashboard** — loading states, error handling, summary statistics, and a results table
- [x] **Per-visitor Spotify sessions** — real per-visitor login (each visitor reads their own playlists through their own session, persisted and refreshed in Postgres), made necessary by Spotify's February 2026 API changes restricting playlist access to the authenticated account's own playlists; plus a no-login "try an example" path behind one curated, owner-held session, for visitors without a Spotify account (see `docs/MVP_ROADMAP.md`, Milestone 7)
- [ ] **Deployment** — containerized backend on Cloud Run, deployed frontend, production database

The application is functionally complete end-to-end — what remains is
deployment.

## How it works

```
User → React frontend → Go API ─┬─→ Spotify Web API   (playlist import)
                                 ├─→ PostgreSQL        (availability cache)
                                 └─→ JOYSOUND          (live availability lookups)
```

1. The visitor logs in with Spotify (OAuth Authorization Code flow). The
   backend persists their session — access token, refresh token, expiry — as
   a row in Postgres, and hands their browser an opaque session ID as a
   cookie. Every later request resolves that cookie back to a fresh,
   automatically-refreshed access token, so each visitor's playlists are
   read through their own account, not a shared one. (A visitor without a
   Spotify account can skip all of this via "try an example," which runs the
   same flow against a curated playlist using one dedicated, owner-held
   session instead.)
2. The visitor submits one of their own playlist URLs, and the backend
   extracts its unique artists using their access token.
3. For each artist, the backend checks a Postgres-backed cache first; on a
   miss or a stale (30-day) entry, it searches JOYSOUND live, rate-limited
   and capped per request to stay considerate of JOYSOUND's servers.
4. The combined results — per-artist availability plus how many were actually
   checked — are returned to the frontend.

## Architecture

A **domain-oriented modular monolith**: a single deployable Go service
organized around business domains (`spotify`, `playlist`, `karaoke`) with
clear interface seams between them, rather than technical layers. The goal is
a structure that's fast to build and easy to reason about now, but doesn't
make it hard to extract a domain into its own service later if it needs to
scale independently. See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for
the full reasoning, including the trade-offs considered.

## Tech stack

| Layer | Technology |
| --- | --- |
| Backend | Go, `net/http`, `pgx`, PostgreSQL, `golang-migrate` |
| Frontend | React, TypeScript, Vite |
| External APIs | Spotify Web API (OAuth), JOYSOUND (HTML scraping) |

## Running locally

**Prerequisites:** Go, Node.js, PostgreSQL (or Docker, via the included
`docker-compose.yml`), and a [Spotify Developer](https://developer.spotify.com/)
app for API credentials.

```bash
# Database
cd backend && docker compose up -d
migrate -path migrations -database "$DATABASE_URL?sslmode=disable" up

# Backend (copy .env.example to .env and fill in Spotify credentials first)
go run ./cmd/api

# Frontend (copy .env.example to .env first)
cd frontend && npm install && npm run dev
```

Both `.env.example` files call out a constraint worth knowing up front: the
frontend origin, backend origin, and Spotify's registered `redirect_uri` all
need to agree on host (`127.0.0.1` vs. `localhost` count as different "sites"
for cookie purposes, and the session cookie won't survive a mismatch).
