# MVP Roadmap

## Goal

Deliver a working application that allows a user to log in with their own Spotify account, paste one of their playlist URLs, and determine which artists are available on JOYSOUND.

The MVP prioritizes validation of the core product idea over completeness.

---

# Milestone 1: Project Setup

## Objective

Create a working development environment.

## Tasks

* Initialize Git repository
* Create React frontend
* Create Go backend
* Setup PostgreSQL
* Configure environment variables
* Establish frontend ↔ backend communication

## Success Criteria

* Frontend can call backend APIs
* Backend can connect to PostgreSQL
* Local development environment is functional

---

# Milestone 2: Spotify Playlist Import

## Objective

Import a public Spotify playlist and extract artist information.

## Tasks

* Accept Spotify playlist URL
* Authenticate with Spotify via the Authorization Code flow (delivered here as
  a single shared backend session — sufficient at the time, since Spotify
  still served public-playlist data to any authenticated app; Milestone 7
  replaces this with real per-visitor sessions once that stopped being true —
  see PROJECT_PLAN.md and ARCHITECTURE.md)
* Retrieve playlist metadata
* Extract songs
* Extract unique artists
* Return results through API

## Success Criteria

User can paste:

```text
https://open.spotify.com/playlist/...
```

and receive:

```text
Architects
Polaris
Bring Me The Horizon
```

---

# Milestone 3: JOYSOUND Integration

## Objective

Determine whether an artist exists on JOYSOUND.

## Tasks

* Implement JOYSOUND client
* Integrate JOYSOUND search results page
* Parse search-results HTML
* Return availability result

## Success Criteria

Example:

```text
Bring Me The Horizon
```

returns:

```text
Available: Yes
```

---

# Milestone 4: Playlist Matching

## Objective

Combine Spotify import and JOYSOUND matching.

## Tasks

* Process all playlist artists
* Query JOYSOUND for each artist
* Aggregate results
* Return match summary

## Success Criteria

Display:

| Artist               | JOYSOUND |
| -------------------- | -------- |
| Architects           | Yes      |
| Thornhill            | No       |
| Bring Me The Horizon | Yes      |

This milestone delivers the first complete version of the product.

---

# Milestone 5: Availability Cache

## Objective

Reduce repeated JOYSOUND lookups.

## Tasks

* Create availability table
* Check cache before JOYSOUND requests
* Store lookup results
* Record last_checked timestamp

## Success Criteria

Repeated searches use cached results when available.

---

# Milestone 6: Results Dashboard

## Objective

Provide a polished user experience.

## Tasks

* Improve UI
* Add loading states
* Add error handling
* Add summary statistics

## Example

```text
24 artists found
18 available on JOYSOUND
75% coverage
```

## Success Criteria

Application is usable without developer knowledge.

---

# Milestone 7: Per-Visitor Spotify Sessions

## Objective

Replace the single shared Spotify session with real per-visitor
authentication, so that "check your own playlists" is true for every visitor
— not just whoever last logged in.

This milestone exists because Spotify's early 2026 Web API changes mean
playlist track data is now only returned for playlists the authenticated
account owns or collaborates on, for any app in Development Mode (see
`ARCHITECTURE.md` and `PROJECT_PLAN.md` for the full account of how this was
discovered and what it replaces).

## Tasks

* Add a session store mapping browser sessions/cookies to Spotify tokens
* Issue, persist, and refresh Spotify tokens per visitor (replacing the
  current single in-memory `userToken`/`refreshToken` pair)
* Rework `Client` and the HTTP handler layer so every request acts on behalf
  of the visitor who made it, not a single cached account
* Add a visible "Login with Spotify" entry point to the frontend
* Add a "try an example" path: keep one deliberate, owner-held session
  (Andrew's own, refreshed like any other) behind a couple of curated
  playlist links on the landing page, so a visitor without a Spotify account
  — e.g. a recruiter — can see the app working end-to-end with one click,
  no login required. This repurposes today's single-session mechanism from
  an accidental default into an intentional demo feature.

## Success Criteria

Two different visitors, logged in with two different Spotify accounts, each
see results for their own playlists — not each other's. A visitor with no
Spotify account at all can still try the app via the example-playlist links.

---

# Milestone 8: Deployment

## Objective

Make the application publicly accessible.

## Tasks

* Containerize backend
* Deploy to Cloud Run
* Deploy frontend
* Configure production database
* Configure environment variables

## Success Criteria

Application is accessible through a public URL.

---

# Out of Scope for MVP

The following features are intentionally deferred:

* Song-level matching
* Persistent user accounts (saved playlists, search history)
* Recommendation engine
* gRPC
* Protocol Buffers
* Pub/Sub
* Terraform
* Kubernetes

These features will be considered after MVP validation.

---

# MVP Definition of Done

A user can:

1. Log in with their own Spotify account and paste one of their playlist URLs
2. Import playlist artists
3. Check artist availability on JOYSOUND
4. View results in a web interface
5. Receive responses backed by a local cache

Once these requirements are met, the MVP is considered complete.
