# KaraokeMatch

## Project Overview

KaraokeMatch helps users determine which songs and artists from their Spotify playlists are available on Japanese karaoke services.

The project is motivated by the difficulty of finding karaoke-available songs for niche genres such as metalcore, post-hardcore, and progressive metal.

The initial focus is on delivering a useful product quickly while gradually evolving the system toward a more scalable, microservice-oriented architecture.

Based on early technical research, the MVP will focus exclusively on DAM integration due to the discovery of a structured JSON search API. JOYSOUND support remains a planned future enhancement.

---

# Problem Statement

When preparing for karaoke, users often spend significant time manually searching DAM and JOYSOUND to determine whether songs from their Spotify playlists are available.

This process becomes increasingly frustrating for users with large playlists or niche music tastes.

KaraokeMatch aims to automate this process.

---

# Goals

## Product Goals

* Import Spotify playlists
* Identify karaoke-available artists and songs
* Reduce manual searching
* Help users discover karaoke-friendly alternatives

## Engineering Goals

* Learn and demonstrate Go development
* Practice backend API design
* Practice automated testing in Go (table-driven tests, idiomatic use of `testing`)
* Build production-style systems
* Learn GCP deployment
* Incrementally introduce microservices, gRPC, and event-driven architecture

---

# MVP Scope

The MVP intentionally prioritizes simplicity and rapid delivery.

The MVP will support DAM only.

JOYSOUND integration is deferred until a reliable and maintainable integration approach is identified.

## User Flow

User logs in with their own Spotify account.

↓

User pastes one of their own Spotify playlist URLs.

↓

Playlist is imported.

↓

Artists are extracted.

↓

Artists are matched against DAM availability.

↓

Results are displayed.

No persistent user accounts required — no signup, profile, or saved history.
A visitor's Spotify login only needs to last for the session.

**Implementation note:** Spotify's API has moved the ground under this flow
twice. First, in late 2024, even reading public playlist data started
requiring the backend to complete an OAuth Authorization Code exchange — at
the time, a one-time backend technical step that stayed invisible to end
users (a single shared session was enough, because Spotify still returned
track data for any playlist to any authenticated app). Then, in their
February 2026 Web API changes, Spotify restricted playlist track data to
playlists the *authenticated account* owns or collaborates on — public or
not. That makes "their own playlists" a literal, per-visitor requirement:
each user must now log in with their own Spotify account for the feature to
work at all. What was an invisible backend step has become the user-facing
"Login with Spotify" feature described in Phase 3 below — see
`ARCHITECTURE.md` for the per-visitor session architecture this requires.

---

# MVP Features

## Spotify Playlist Import

Users log in with their own Spotify account, then paste:

https://open.spotify.com/playlist/...

The application extracts:

* Playlist name
* Artists
* Songs

Requirements:

* The user must be logged in with their own Spotify account
* The playlist must be one the user owns or collaborates on — per Spotify's
  February 2026 Web API restrictions, track data is no longer available for
  other playlists, public or not

---

## Artist-Level Matching

Initial matching occurs at the artist level.

The application checks whether artists from the playlist are available on DAM.

Example:

Playlist contains:

* Architects
* Polaris
* Bring Me The Horizon

Results:

| Artist               | DAM |
| -------------------- | --- |
| Architects           | Yes |
| Polaris              | No  |
| Bring Me The Horizon | Yes |

This approach significantly reduces complexity while still providing useful results.

---

## DAM Integration

Research has identified a publicly accessible DAM search API that returns structured JSON responses.

Example response:

```json
{
  "data": {
    "totalCount": 48
  },
  "list": [
    {
      "artist": "Bring Me The Horizon",
      "title": "Kingslayer feat. BABYMETAL"
    }
  ]
}
```

This allows the MVP to avoid HTML scraping and instead consume structured search results.

Benefits:

* Faster implementation
* More reliable matching
* Simpler backend logic
* Easier future expansion to song-level matching

---

## Karaoke Availability Cache

MVP uses a lightweight cached catalog approach.

### Hybrid Cache Strategy

Workflow:

1. Check local database
2. If artist exists, return cached result
3. If artist does not exist:

   * Query DAM API
   * Store result
   * Return result

Benefits:

* Faster than querying DAM every request
* Reduced external requests
* Easier than maintaining a complete catalog
* Catalog grows naturally over time

### Cache Freshness

Each record stores:

* artist_name
* available
* last_checked

Future versions may refresh stale records automatically after a configurable TTL period.

---

## Results Dashboard

Display:

* Available artists
* Unavailable artists
* DAM coverage percentage
* Summary statistics

Example:

* 14 artists available
* 7 artists unavailable
* 67% DAM coverage

---

# MVP Technical Stack

## Frontend

### React

Purpose:

* User interface
* Playlist input
* Results dashboard

### TypeScript

Purpose:

* Type safety
* Better maintainability

---

## Backend

### Go

Primary backend language.

Responsibilities:

* Spotify integration
* Matching logic
* API endpoints
* DAM integration
* Cache management

---

## Database

### PostgreSQL

Stores:

#### Artists

* artist_id
* artist_name

#### Availability Cache

* artist_name
* available
* last_checked

#### Playlists (optional)

* playlist_id
* spotify_playlist_id

---

## Deployment

### Google Cloud Run

Host Go backend.

Benefits:

* Simple deployment
* Minimal infrastructure overhead
* Good learning experience

---

# Architecture

## MVP Architecture

User

↓

React Frontend

↓

Go API

↓

PostgreSQL Cache

↓

DAM Search API

Simple monolith.

No microservices initially.

---

# Future Enhancements

## Phase 2: Song-Level Matching

Move beyond artist availability.

Leverage DAM song metadata returned by the search API.

Example:

Instead of:

"Bring Me The Horizon available"

Show:

| Song                  | DAM |
| --------------------- | --- |
| Kingslayer            | Yes |
| Throne                | Yes |
| Can You Feel My Heart | Yes |

Benefits:

* Much more useful for karaoke planning
* More accurate availability information
* More interesting matching challenges

---

## Phase 3: Persistent Accounts

Per-visitor "Login with Spotify" and access to one's own playlists are now
part of the MVP itself (Milestone 7 in `MVP_ROADMAP.md`) — Spotify's
February 2026 API changes made that a prerequisite for the core feature
rather than a future nice-to-have. What remains a deferred enhancement is
*persisting* that beyond a single session:

* Save previous imports
* Search history across visits

Additional tables:

* saved_playlists
* search_history

---

## Phase 4: JOYSOUND Integration

Add support for JOYSOUND.

Current investigation suggests:

* Search pages are accessible
* Search results can be rendered server-side
* A structured API has not yet been identified

Potential implementation approaches:

* Discover internal API endpoints
* Parse search responses
* Parse React/Next.js response payloads

Result format:

| Artist     | DAM | JOYSOUND |
| ---------- | --- | -------- |
| Architects | Yes | Yes      |
| Polaris    | No  | Yes      |

---

## Phase 5: Recommendation Engine

If a song is unavailable:

Recommend:

* Similar artist
* Similar song
* Popular karaoke alternative

Example:

Unavailable:

* Currents

Suggested:

* Crossfaith
* Coldrain
* SiM

---

## Phase 6: Microservice Architecture

Split monolith into services.

### API Service

Responsibilities:

* Authentication
* User APIs
* Playlist APIs

### Matching Service

Responsibilities:

* Artist matching
* Song matching
* Recommendations

Communication:

* Protocol Buffers
* gRPC

This phase aligns closely with Mercari's architecture.

---

## Phase 7: Event-Driven Processing

Introduce asynchronous processing.

### Google Pub/Sub

Workflow:

Playlist Imported

↓

Publish Event

↓

Matching Service Consumes Event

↓

Process Songs

↓

Store Results

Benefits:

* Scalability
* Decoupling
* Demonstrates distributed systems concepts

---

## Phase 8: Monitoring & Observability

### OpenTelemetry

Collect:

* Request latency
* Error rates
* Throughput

### Grafana

Visualize:

* Service health
* API performance
* Matching success rates

---

## Phase 9: Infrastructure as Code

### Terraform

Manage:

* Cloud Run
* Cloud SQL
* Service Accounts
* Pub/Sub Topics

---

## Phase 10: Kubernetes

Move from Cloud Run to:

Google Kubernetes Engine (GKE)

Demonstrates:

* Container orchestration
* Horizontal scaling
* Production-grade deployment

---

# Success Metrics

## Product Metrics

* Playlist import success rate
* Artist match accuracy
* Song match accuracy
* User retention

## Engineering Metrics

* API latency
* Error rate
* Deployment frequency
* Service uptime

---

# Technologies Roadmap

## MVP

* React
* TypeScript
* Go
* PostgreSQL
* Cloud Run
* DAM Search API

## Intermediate

* Spotify OAuth
* Docker
* GitHub Actions

## Advanced

* gRPC
* Protocol Buffers
* Pub/Sub
* OpenTelemetry
* Grafana
* Terraform

## Production-Style

* Kubernetes (GKE)
* Distributed services
* Recommendation engine
* Analytics pipeline

---

# Why This Project

This project demonstrates:

* Backend engineering
* API development
* Database design
* Cloud deployment
* External service integration
* Caching strategies
* Scalability considerations
* Product thinking
* Real-world user value

while providing a clear pathway toward technologies commonly used in modern large-scale backend systems.
