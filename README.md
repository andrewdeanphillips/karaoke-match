# KaraokeMatch

KaraokeMatch tells you which artists from a Spotify playlist are actually
available on [JOYSOUND](https://www.joysound.com/), a major Japanese karaoke
platform. Paste a public playlist URL and get back a per-artist breakdown of
what JOYSOUND carries — built to solve a real annoyance: manually
cross-checking niche-genre playlists (metalcore, post-hardcore, prog metal)
against karaoke catalogs one search at a time.

This is also a learning project — a hands-on way to build production-style
backend systems in Go, going deep on API design, database schema design,
caching, and automated testing rather than skimming the surface of a
tutorial.

## Project status

**MVP in progress — 5 of 7 milestones complete.** See
[`docs/MVP_ROADMAP.md`](docs/MVP_ROADMAP.md) for the full plan.

- [x] **Project setup** — Go backend, React/TypeScript frontend, PostgreSQL, CORS wiring between them
- [x] **Spotify playlist import** — OAuth Authorization Code flow, playlist parsing, unique-artist extraction
- [x] **JOYSOUND integration** — search-results scraping and artist matching, including handling same-name collisions across catalogs
- [x] **Playlist matching** — a single endpoint combining playlist import with per-artist availability checks
- [x] **Availability cache** — a Postgres-backed cache (with TTL) sitting in front of JOYSOUND, plus a rate limiter and a request budget that bounds how many live lookups one playlist match can trigger
- [ ] **Results dashboard** — frontend polish: loading states, error handling, summary statistics
- [ ] **Deployment** — containerized backend on Cloud Run, deployed frontend, production database

The backend and its API are functionally complete for the MVP; what remains
is mostly front-end work to turn the working API into a usable product.

## How it works

```
User → React frontend → Go API ─┬─→ Spotify Web API   (playlist import)
                                 ├─→ PostgreSQL        (availability cache)
                                 └─→ JOYSOUND          (live availability lookups)
```

1. The user submits a public Spotify playlist URL.
2. The backend authenticates with Spotify (Authorization Code flow — a
   backend requirement for reading playlist data, not a user-facing login)
   and extracts the playlist's unique artists.
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

# Frontend
cd frontend && npm install && npm run dev
```
