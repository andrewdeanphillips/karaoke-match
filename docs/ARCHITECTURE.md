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
* Public playlists → Spotify OAuth
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
* Spotify OAuth

No architectural changes required.

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
