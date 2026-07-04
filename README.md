# KaraokeMatch

**[Try it live →](https://karaoke-match-723385085873.asia-northeast1.run.app/)**
(no Spotify account needed — the landing page has a one-click example path)

*Built in three days (7–9 June 2026) with Claude as pair programmer — see [AI-assisted development](#ai-assisted-development).*

KaraokeMatch tells you which artists from a Spotify playlist are actually
available on [JOYSOUND](https://www.joysound.com/), a major Japanese karaoke
platform. Log in with Spotify, point it at one of your playlists, and get
back a per-artist breakdown of what JOYSOUND carries — built to solve a real
annoyance: manually cross-checking niche-genre playlists (metalcore,
post-hardcore, prog metal) against karaoke catalogs one search at a time.

This is also a learning project — a hands-on way to build production-style
backend systems in Go, going deep on API design, database schema design,
caching, and automated testing rather than skimming the surface of a
tutorial.

## AI-assisted development

The project was built over three days (7–9 June 2026) with Claude, using a
deliberately designed collaboration framework — [`docs/MVP_PROMPT.md`](docs/MVP_PROMPT.md)
— written before the first line of code. It enforced: **explain-before-code**
(no implementation without a prior plan, waiting for approval), **step-by-step
verification gates** (stop after each step, wait for browser confirmation
before continuing), and **teaching mode** (every new Go concept explained with
Java comparisons, since that's the background I was coming from). Claude acted
as pair programmer and mentor; I acted as product owner, architect, and QA.

The plan changed three times due to real-world discoveries — each one driven by
something I noticed during testing, not something the model predicted:

| Original plan | What actually happened | Why |
|---|---|---|
| Client Credentials auth for Spotify | Full Authorization Code OAuth flow | Spotify had changed their API requirements; discovered live when the first token attempt returned 401 |
| DAM karaoke catalog integration | JOYSOUND instead, and the whole DAM implementation scrapped | Live testing found an uncorrectable false-positive; pausing to audit the approach surfaced a legal grey area too; JOYSOUND resolved both at once |
| "Paste any public playlist URL" | Per-visitor OAuth sessions, each visitor reads their own playlists | Spotify's early 2026 API policy change restricted playlist track access to the playlist owner's account — discovered mid-Milestone 6 |

What broke, what I caught, and what we decided — for these pivots and a couple
of other turning points (a three-round cookie-debugging chain, a production-only
database bug) — is written up in
[`docs/DEVELOPMENT_STORY.md`](docs/DEVELOPMENT_STORY.md).

## Project status

**MVP complete — all 8 milestones shipped and live in production.** See
[`docs/MVP_ROADMAP.md`](docs/MVP_ROADMAP.md) for the full plan.

- [x] **Project setup** — Go backend, React/TypeScript frontend, PostgreSQL, CORS wiring between them
- [x] **Spotify playlist import** — OAuth Authorization Code flow, playlist parsing, unique-artist extraction
- [x] **JOYSOUND integration** — search-results scraping with song- and artist-level matching, including handling same-name collisions across catalogs
- [x] **Playlist matching** — a single endpoint combining playlist import with per-track availability checks
- [x] **Song-level matching** — each track resolves an independent song link and artist link, with an artist-name fallback search when the song title search alone doesn't surface the artist
- [x] **Availability cache** — Postgres-backed cache with TTL in front of JOYSOUND, plus a rate limiter and per-request lookup budget
- [x] **Results dashboard** — loading states, error handling, summary statistics, results table
- [x] **Per-visitor Spotify sessions** — each visitor reads their own playlists via their own persisted, auto-refreshed session (required by Spotify's early 2026 API restrictions), plus a no-login "try an example" path
- [x] **Deployment** — single image (Go binary embedding the built frontend) on Cloud Run, backed by managed PostgreSQL

## How it works

```
User → React frontend → Go API ─┬─→ Spotify Web API   (playlist import)
                                 ├─→ PostgreSQL        (availability cache)
                                 └─→ JOYSOUND          (live availability lookups)
```

1. The visitor logs in with Spotify (OAuth Authorization Code flow). The
   backend persists their session in Postgres and hands the browser an opaque
   session-ID cookie, so every later request reads *their* playlists through
   *their* auto-refreshed token — not a shared account. (No Spotify account?
   "Try an example" runs the same flow against a curated playlist via one
   owner-held session.)
2. The visitor submits one of their playlist URLs; the backend extracts its
   tracks (artist + title).
3. Each track is checked against the Postgres cache first (entries stay fresh
   for 30 days). On a miss it searches JOYSOUND live by song title — one
   combined search page returns both song and artist results, so it usually
   resolves both links in a single lookup; a second search by artist name
   only runs as a fallback when the artist doesn't surface on the title
   search. This is paced to ≥200ms between requests and capped at 50 lookups
   per check, since JOYSOUND has no public API and this means one page load
   per search. Mostly-cached playlists return near-instantly; a cold playlist
   past the 50-lookup cap returns a partial result, and the UI reports how
   many tracks were checked.
4. Results return to the frontend as a Song | Artist table — each track's
   song title and artist name link independently to their JOYSOUND pages
   when a match was found, so a song match and an artist match are both
   visible even when only one of them exists for a track.

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
| Deployment | Docker, Google Cloud Run, managed PostgreSQL |

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
