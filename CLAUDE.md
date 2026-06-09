# Claude Code project instructions

This file establishes the working norms and context for Claude Code sessions
on KaraokeMatch. These rules have been in place since the project's start and
are derived from the collaboration framework in `docs/MVP_PROMPT.md`.

## Workflow

- Follow `docs/MVP_ROADMAP.md` one milestone at a time. Do not skip ahead.
- **Explain the plan before writing any code.** Wait for explicit approval
  before implementing.
- Break work into small steps. Stop after each step and ask for browser
  verification before continuing.
- When the task is a question or exploration ("what could we do about X?"),
  answer in 2–3 sentences with a recommendation and the main tradeoff — do
  not implement until approved.

## Go and React concepts

- Andrew is an experienced Java and JavaScript developer, new to Go.
- Introduce Go concepts as they come up; compare to Java equivalents.
- Frame frontend changes as "modern React practices" and flag any deprecated
  API usage (`FormEvent` vs `SubmitEvent`, etc.).

## Code style

- Prefer simple, idiomatic code. No premature abstractions.
- Default to **no comments**. Only add a comment when the *why* is
  non-obvious: a hidden constraint, a subtle invariant, a specific bug
  workaround. If removing the comment wouldn't confuse a future reader,
  don't write it.
- **Comments must not name-check identifiers from a different part of the
  codebase** — including cross-stack references (a TypeScript comment must
  not name a Go function, and vice versa). Same rot-risk as citing a doc
  file by name, worse because it requires bilingual fluency to catch when
  the reference goes stale. Commit messages are exempt — they are historical
  "why we did this then" snapshots, not living documentation.
- Never add error handling, fallbacks, or validation for scenarios that
  cannot happen. Trust framework guarantees. Only validate at system
  boundaries.

## Architecture

- Domain-oriented modular monolith. Domains: `spotify`, `playlist`,
  `karaoke`. Each owns its `models.go`, `service.go`, `handler.go`.
  Cross-domain orchestration belongs in `cmd/api` (the composition root),
  not inside a domain.
- Frontend: `pages/` own page-level state; `components/` are self-contained
  and reusable.
- Avoid introducing microservices, gRPC, Pub/Sub, Terraform, Kubernetes, or
  other future-phase technologies unless explicitly requested.
- `docs/private/` and `docs/journal/` are gitignored — local only. Read
  them directly when needed; never commit their paths into source.

## Git

- Suggest commit points. Keep commits small and focused.
- Prefer creating a new commit to amending, unless explicitly asked to amend.
- Never skip hooks (`--no-verify`) or bypass signing without explicit
  instruction.
- Use feature branches for new work when a PR in the history is useful
  (e.g. post-MVP features). The JOYSOUND artist links feature was the first
  branch; use the same pattern for subsequent post-MVP work.

## UI verification

- No browser tool is available in this environment. For UI changes: confirm
  the dev servers are running, then use `AskUserQuestion` to have Andrew
  verify in the browser before proceeding.
- Dev servers: backend on `:8080` (log: `/tmp/karaoke-backend.log`),
  frontend on `:5173` (log: `/tmp/karaoke-frontend.log`).

## Session journals

- One journal entry per session in `docs/journal/NN-description.md`.
  Gitignored, local only. Written at the end of the session by whoever has
  the most complete context of what happened.
- Format: Milestone / What we built / concepts introduced / decisions made /
  bugs hit / next up. See existing entries for tone and length.

