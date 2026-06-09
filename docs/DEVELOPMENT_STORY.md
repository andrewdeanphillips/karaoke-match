# Development story

KaraokeMatch was built over three days (7–9 June 2026) with Claude as a pair
programmer and technical mentor. This document captures the four moments that
most clearly show how that collaboration worked — specifically, the moments
where something unexpected happened and a decision had to be made. In each
case the interesting part isn't "the AI wrote code"; it's the interplay
between what the model predicted and what reality turned out to be.

The full detail is in `docs/journal/` (gitignored, local only). This document
is the compressed version. Claude wrote the journals and this summary from the
build transcripts; the decisions described are mine. Where the code involved
shipped, it's cited by path
so the claims are checkable against the repo — and where it didn't (the DAM
work in §1 was scrapped before any commit), that's called out too.

---

## 1. The false positive that was predicted before it happened

The first karaoke catalog target was DAM. Before writing any matching logic, I
raised a concern: DAM's search returns results for both artist names and song
titles, so checking `totalCount > 0` would produce false positives. We built
`anySongCreditsArtist` specifically to guard against this — comparing each
result's artist field, not just the count.

Then live testing against real data found something neither of us had
anticipated: searching "Polaris" (an Australian metalcore band I actually
listen to) returned `available: true` — correctly by the letter of the
matching code, and wrong by its intent. DAM's catalog contains a completely
unrelated act also literally named "Polaris." An exact, case-insensitive match
can still match the wrong same-named artist. No string-matching refinement
fixes this; it requires a stable per-artist identifier that DAM doesn't
expose.

The "guard we built first" caught the title problem; the live test found the
identity problem. Neither showed up during planning.

### The legal detour

Right before committing that DAM work, I stopped to reconsider: was calling
a reverse-engineered internal API actually appropriate for a portfolio project?
Rather than guessing, we investigated: DAM's search page ships an empty
`<ul>`, and its own JavaScript bundle builds and fires the exact request we'd
reverse-engineered, using credentials it ships in plaintext to every browser
visitor. "Scrape the page" and "call the API" are the same action here — a
headless scraper would just be a slower, heavier way to trigger an identical
call with identical credentials.

That was the honest answer, and it pointed toward a separate question: was
there a cleaner alternative that would solve both problems at once? That's
when we checked JOYSOUND.

JOYSOUND's search page is server-rendered HTML containing results directly,
with a stable numeric `artistId` per artist embedded in the markup — exactly
the stable identifier DAM never had. That identifier is what the cache keys
repeat lookups on today (the `catalog_artist_id` column read in
`backend/internal/karaoke/repository.go`). Its `robots.txt` explicitly permits
the search route for general crawlers. DAM had neither. JOYSOUND resolved the
legal, false-positive, and identifier concerns at once. Decision: JOYSOUND in
scope, DAM out. Nothing DAM-related was ever committed — the git history
contains no DAM code, which is the cleanest evidence the swap landed before any
of it shipped.

The appropriateness question was mine, not the model's: it didn't raise it
unprompted. Checking the assumption before committing is what turned "document
the limitation and move on" into a better catalog choice.

---

## 2. A platform policy change invalidated the MVP premise mid-build

Milestone 6 was "Results Dashboard" — the frontend. During testing, I noticed
the backend could read my own playlists but returned 403/404 for playlists
owned by other people.

The MVP's stated goal, written at the project's start, was literally: "Deliver
a working application that allows a user to paste a **public** Spotify playlist
URL." I asked whether that was expected behavior.

Research confirmed: Spotify rolled out a Web API change in early 2026
restricting `GET /playlists/{id}/items` for apps in Development Mode to
playlists the authenticated user owns or collaborates on. Other playlists —
even fully public ones — return 403 or simply omit the `items` field. The
restriction had been in place for months before the project started — it just
wasn't discovered until testing in Milestone 6.

We ruled out three workarounds:
- **Client Credentials flow** — Spotify states it can't fetch playlist items
  under the same restriction, regardless of token type
- **Scraping the web player** — JS-rendered, hashed CSS class names that
  change on every deploy, probable ToS violation
- **Applying for Extended Quota Mode** — exempt from the restriction, but
  gated on Spotify's approval process with no guaranteed timeline

The decision: pivot to real per-visitor OAuth — each visitor authenticates
with their own Spotify account and the backend reads *their* playlists through
*their* token. "Check whether your playlists would work at a Japanese karaoke
bar" is still a coherent product idea, and it's one that actually works.

This pivot also surfaced something neither of us had noticed: the backend had
no per-visitor session at all. Every request, from every browser, used
whichever token happened to be in memory — a single shared field on the
`Client` struct. "Your own playlists" was, at that moment, only true for me.
Building it properly required a full session architecture: a new
`spotify_sessions` Postgres table (`backend/migrations/000002_create_spotify_sessions.up.sql`),
per-session token issuance and refresh (`backend/internal/spotify/session.go`),
reworked middleware, and a cookie-based session ID.

That became Milestone 7 — a whole new milestone inserted mid-project to
address a constraint the plan was written without.

---

## 3. The cookie debugging chain

Getting the per-visitor session cookie to actually work in the browser took
three separate rounds of "wait, why didn't that work" — all the same
underlying rule wearing different clothes.

**Round 1**: After OAuth login, the callback rejected the state parameter with
"invalid or missing." The frontend's login link pointed at `localhost:8080`,
so the CSRF state cookie got set on the `localhost` site — but the registered
Spotify redirect URI is `127.0.0.1:8080/callback`, a different site.
Browsers treat `localhost` and `127.0.0.1` as distinct. The CSRF cookie never
came back.

**Round 2**: Login appeared to succeed but the app still showed the login
prompt. Same root cause, one layer higher — the session cookie was set on
`127.0.0.1` (where `/callback` lives) but the frontend was loading from
`localhost:5173`, a different site. The session cookie was never sent.

**Round 3**: Pointing everything at `127.0.0.1` broke the Vite dev server,
which was only listening on `[::1]` (IPv6 loopback). `127.0.0.1` and `[::1]`
are different addresses; `localhost` happened to resolve to `[::1]` while
`127.0.0.1` didn't. Fix: `server.host: '127.0.0.1'` in `vite.config.ts`.

None of this matters in production once real domains are involved. But the
three rounds clarified something worth knowing explicitly: cookies carry a
"site" boundary, and every origin in the system — frontend URL, backend URL,
Spotify redirect URI — must agree on what "site" means, or cookies silently
disappear. That lesson fed directly into the Milestone 8 deployment design:
serving frontend and backend behind the same origin (via `embed.FS` embedding
the built frontend directly into the Go binary — `backend/cmd/api/web.go`)
rather than as two separate services, which would have reintroduced exactly
this class of problem on real domains with real traffic.

---

## 4. A production bug that couldn't have been caught locally

After the first successful production deployment, verifying the "try an
example" flow returned an intermittent 500 with a database error in the logs:

```
prepared statement "stmtcache_..." already exists (SQLSTATE 42P05)
```

Re-running the exact same request seconds later returned a clean 200. That
flip-flop is usually a sign to distrust the error — except here, the
inconsistency itself was the diagnostic.

`pgx` (the Go Postgres driver) names prepared statements by hashing their SQL
text and caches them per logical connection — a reasonable optimization against
a direct Postgres connection. The production `DATABASE_URL` runs through
Supabase's Supavisor connection pooler in *transaction mode*, which can hand
the same logical client connection a different physical backend connection
between transactions. `pgx` sometimes finds itself on a backend connection
that another multiplexed client already prepared an identically-named statement
on — collision. The error fires only when that happens to be the physical
connection assigned, which is why it was intermittent.

It would have kept firing at random — on logins, cache lookups, playlist
matches, any database operation — more often as real traffic grew.

The fix touched no source — only the deploy-time `DATABASE_URL`: appending
`default_query_exec_mode=simple_protocol` tells `pgx` to skip named prepared
statements entirely and use Postgres's simple query protocol, the standard
documented way to run `pgx` behind PgBouncer or Supavisor in transaction-pooling
mode. (Nothing to cite in the repo for this one — it lives in the production
environment config, not the committed code.)

It only manifests once the real production topology — the real pooler, the real
multiplexing — is in the loop, so local testing could never have surfaced it.
The diagnostic move was treating the intermittency as a signal rather than a
fluke to retry past.
