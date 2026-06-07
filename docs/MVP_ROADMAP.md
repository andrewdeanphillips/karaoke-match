# MVP Roadmap

## Goal

Deliver a working application that allows a user to paste a public Spotify playlist URL and determine which artists are available on DAM.

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

# Milestone 3: DAM Integration

## Objective

Determine whether an artist exists on DAM.

## Tasks

* Implement DAM client
* Integrate discovered DAM search API
* Parse API responses
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

Combine Spotify import and DAM matching.

## Tasks

* Process all playlist artists
* Query DAM for each artist
* Aggregate results
* Return match summary

## Success Criteria

Display:

| Artist               | DAM |
| -------------------- | --- |
| Architects           | Yes |
| Polaris              | No  |
| Bring Me The Horizon | Yes |

This milestone delivers the first complete version of the product.

---

# Milestone 5: Availability Cache

## Objective

Reduce repeated DAM lookups.

## Tasks

* Create availability table
* Check cache before DAM requests
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
18 available on DAM
75% coverage
```

## Success Criteria

Application is usable without developer knowledge.

---

# Milestone 7: Deployment

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

* JOYSOUND integration
* Song-level matching
* Spotify OAuth
* User accounts
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

1. Paste a public Spotify playlist URL
2. Import playlist artists
3. Check artist availability on DAM
4. View results in a web interface
5. Receive responses backed by a local cache

Once these requirements are met, the MVP is considered complete.
