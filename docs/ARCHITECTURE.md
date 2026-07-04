# Architecture

## Architectural Approach

KaraokeMatch uses a **Domain-Oriented Modular Monolith** architecture.

The application is organized around business domains rather than technical layers. This provides a good balance between rapid MVP development and future extensibility.

The MVP remains a single deployable service while maintaining clear boundaries that can later be extracted into separate services if required.

---

## Why This Architecture

### Fast MVP Development

A single Go application is significantly simpler to build, deploy, and debug than a microservice architecture.

This allows focus on validating the product idea rather than infrastructure.

### Easy to Change

The most likely future changes are:

* JOYSOUND → JOYSOUND + additional karaoke catalogs
* Artist matching → Song matching
* Per-visitor sessions → saved imports and search history across visits
  (the per-visitor OAuth sessions already shipped; persisting their results
  is the remaining step)
* Synchronous processing → Background jobs
* Monolith → Microservices

A domain-oriented structure makes these changes easier to implement without large refactors.

### Natural Evolution Path

This is a learning project, and the architecture reflects that. The
domain-oriented modular monolith keeps the application simple to build and
reason about while practising the boundary-drawing and interface-design
thinking that applies at any scale. The `catalog`, `cache`, and `limiter`
interfaces are shaped by how a real microservice would expose its contracts —
not because extraction is planned, but because designing to that shape is part
of what makes the architecture worth building.

The Future Evolution section below sketches what further growth would look
like — as much for the learning value of thinking through the patterns as for
any real operational need.

---

# High-Level Architecture

User

↓

React Frontend

↓

Go API

↓

PostgreSQL / Spotify Web API / JOYSOUND

The application is deployed as a single monolithic service on Cloud Run.

---

# Folder Structure

```text
backend/
├── cmd/
│   └── api/
│       ├── main.go
│       ├── match.go
│       ├── examples.go
│       ├── spotify_auth.go
│       └── web.go
│
├── internal/
│   ├── playlist/
│   │   ├── service.go
│   │   └── models.go
│
│   ├── spotify/
│   │   ├── client.go
│   │   ├── session.go
│   │   ├── context.go
│   │   └── models.go
│
│   ├── karaoke/
│   │   ├── service.go
│   │   ├── joysound_client.go
│   │   ├── repository.go
│   │   └── models.go
│
│   └── database/
│       └── postgres.go
│
├── migrations/
├── Dockerfile
└── go.mod
```

```text
frontend/
├── src/
│   ├── pages/
│   ├── components/
│   ├── api/
│   └── types/
```

---

# Domain Responsibilities

## Playlist

Responsible for:

* Playlist import
* Playlist processing
* Artist extraction

## Spotify

Responsible for:

* Spotify API integration
* Playlist retrieval
* Metadata normalization
* Per-visitor session and token management (login, persistence, refresh —
  shipped in the MVP; see Future Evolution, Phase 1)

## Karaoke

Responsible for:

* JOYSOUND integration (scraping, song- and artist-level matching with fallback)
* Availability lookup and result aggregation
* Caching strategy

This is expected to become the largest domain and a potential future microservice.

## Database

Responsible for:

* PostgreSQL connections
* Database configuration
* Migrations

---

# Testing Approach

KaraokeMatch uses Go's built-in `testing` package — no external test
frameworks or assertion libraries are needed at this scale.

## What gets tested

Tests focus on **domain logic with real value to verify**:

* Artist extraction from playlist data
* JOYSOUND search result parsing and artist matching
* Availability lookup and result aggregation
* Cache read/write behavior and TTL freshness logic

Trivial code — simple struct wiring, thin handlers that just delegate to a
service, configuration loading — is left untested. The goal is meaningful
coverage of logic that can actually break, not 100% coverage for its own sake.

## Conventions

* Test files live alongside the code they test, named `xxx_test.go` (Go
  convention — `go test ./...` discovers them automatically)
* Prefer **table-driven tests** for functions with multiple input/output
  cases — Go's idiomatic alternative to parameterized tests
* Use Go interfaces to substitute fakes for external dependencies (e.g., a
  fake JOYSOUND client when testing matching logic), avoiding the need for a
  mocking framework
* `net/http/httptest` is used to test HTTP handlers without running a server

## Why this approach

Tests are written **alongside the logic they verify**, not as a separate
phase or a strict test-first (TDD) discipline. This keeps the focus on
learning Go's idioms first, while still building the habit of testing the
logic that matters — demonstrating a practical, production-style approach
to testing without overengineering process for an MVP.

---

# Future Evolution

### Phase 1 (MVP) — shipped

```text
Frontend
↓ (session cookie)
Go API
↓
Session Store (Postgres) ←→ Spotify (per-visitor tokens)
↓
JOYSOUND
```

Delivered: playlist matching, JOYSOUND availability checks, Postgres-backed
availability cache, per-visitor Spotify sessions, no-login demo path, Cloud
Run deployment.

### Phase 2 — song-level matching ✓ shipped (post-MVP)

Each track resolves an independent song link and artist link from one
combined JOYSOUND search (by song title), with a second search by artist
name as a fallback when the artist doesn't surface on the title search.
Backed by a dedicated song-availability cache table alongside the existing
artist one.

Remaining from the original Phase 2 idea: additional karaoke catalog sources
alongside JOYSOUND.

### Phase 3

The `catalog` and `cache` interface boundaries are the natural extraction
point if the Karaoke domain were ever split into its own service:

```text
Frontend
↓
API Service
↓ gRPC
Karaoke Service
↓
PostgreSQL
```

### Phase 4

Asynchronous processing via Pub/Sub would decouple playlist import from
availability checking:

```text
Playlist Imported
↓
Pub/Sub
↓
Karaoke Service
↓
Store Results
```
