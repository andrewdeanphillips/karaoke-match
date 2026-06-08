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

* DAM → DAM + JOYSOUND
* Artist matching → Song matching
* Public playlist access → Private playlist access + saved imports (extending
  the OAuth mechanism already required for reading public playlists with
  broader scopes and user accounts)
* Synchronous processing → Background jobs
* Monolith → Microservices

A domain-oriented structure makes these changes easier to implement without large refactors.

### Natural Evolution Path

The Karaoke domain can later become its own service using:

* Protocol Buffers
* gRPC
* Pub/Sub

This aligns with technologies commonly used in large-scale backend systems while avoiding unnecessary complexity during the MVP stage.

---

# High-Level Architecture

User

↓

React Frontend

↓

Go API

↓

PostgreSQL

↓

DAM Search API

The application is initially deployed as a single monolithic service on Cloud Run.

---

# Folder Structure

```text
backend/
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│
│   ├── playlist/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── models.go
│
│   ├── spotify/
│   │   ├── client.go
│   │   └── models.go
│
│   ├── karaoke/
│   │   ├── service.go
│   │   ├── dam_client.go
│   │   ├── joysound_client.go
│   │   ├── matcher.go
│   │   ├── repository.go
│   │   └── models.go
│
│   ├── database/
│   │   └── postgres.go
│
│   └── shared/
│       └── types.go
│
├── migrations/
├── Dockerfile
└── go.mod
```

Frontend:

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
  see Future Evolution, Phase 2)

## Karaoke

Responsible for:

* DAM integration
* JOYSOUND integration
* Matching logic
* Availability lookup
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

* Artist/song extraction from playlist data
* DAM API response parsing
* Matching logic (availability lookups, result aggregation)
* Cache read/write behavior

Trivial code — simple struct wiring, thin handlers that just delegate to a
service, configuration loading — is left untested. The goal is meaningful
coverage of logic that can actually break, not 100% coverage for its own sake.

## Conventions

* Test files live alongside the code they test, named `xxx_test.go` (Go
  convention — `go test ./...` discovers them automatically)
* Prefer **table-driven tests** for functions with multiple input/output
  cases — Go's idiomatic alternative to parameterized tests
* Use Go interfaces to substitute fakes for external dependencies (e.g., a
  fake DAM client when testing matching logic), avoiding the need for a
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

### Phase 1 (MVP)

Monolith

```text
Frontend
↓
Go API
↓
PostgreSQL
↓
DAM
```

### Phase 2

Add:

* Song matching
* JOYSOUND support
* Per-visitor Spotify sessions — Spotify's February 2026 API changes mean
  playlist data is now only available for playlists the authenticated
  account owns or collaborates on, so this is no longer an optional
  enhancement but a prerequisite for the core feature to work for anyone but
  the app's own developer. Unlike the other items in this phase, this one
  *does* require an architectural change: a session store mapping browser
  sessions to Spotify tokens, replacing the single shared in-memory session
  the MVP launched with.

```text
Frontend
↓ (session cookie)
Go API
↓
Session Store (Postgres) ←→ Spotify (per-visitor tokens)
↓
DAM / JOYSOUND
```

### Phase 3

Extract Karaoke domain into a separate service.

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

Introduce asynchronous processing.

```text
Playlist Imported
↓
Pub/Sub
↓
Karaoke Service
↓
Store Results
```

This allows the architecture to grow incrementally while preserving a working and maintainable MVP.
