# youtube-go-mcp — build plan

Working board for turning this repo (seeded from [raitonoberu/ytmusic](https://github.com/raitonoberu/ytmusic)) into a **static Go** YouTube Music MCP that an AI agent can use with Cast / Nest (or similar playback bridges).

**Related:** a Cast companion example is [mcp-beam](https://github.com/shotah/mcp-beam). This MCP sources tracks; Cast (or another player MCP) plays them.

---

## Current state

- [x] Seed repo with ytmusic Go client (search, watch playlist, lyrics, suggestions)
- [x] Rebrand module path away from `github.com/raitonoberu/ytmusic` → `github.com/shotah/youtube-go-mcp`
- [x] Bump Go version (seed is `go 1.16`; target 1.22+)
- [x] Refresh InnerTube `clientVersion` / headers (seed last touched ~2024 — expect breakage)

---

## Phase 0 — Repo hygiene

- [x] `go.mod` module rename + tidy
- [x] Split packages: keep client as `ytmusic/` (or `internal/ytmusic`), MCP under `cmd/youtube-go-mcp` + `internal/mcp`
- [x] README rewrite (this is an MCP + client, not just a search lib)
- [x] `.gitignore` for auth artifacts (`headers.json`, cookies, tokens)
- [x] CI: `go test ./...`, golangci-lint, GoReleaser on `v*` tags
- [x] Make / Dockerfile for static binary (`CGO_ENABLED=0`)

---

## Phase 1 — Auth (Premium session)

Premium rides on **your Google account session**, not a special Music API key.

- [x] `BrowserAuth` from exported cookies / headers JSON (`cookie` + `x-goog-authuser`)
- [x] Compute `SAPISIDHASH` Authorization header (same model as Python `ytmusicapi`)
- [x] Attach auth to all InnerTube requests when configured
- [x] Clear errors: `AuthRequired`, `InvalidAuth`, expired cookie guidance
- [x] One-shot auth helper CLI: `youtube-go-mcp auth` (print instructions + validate headers)
- [x] Document cookie export flow (browser DevTools → music.youtube.com) — never commit secrets
- [x] Optional: env `YTMUSIC_HEADERS_PATH` / mount path (e.g. `secrets/ytmusic/headers.json`)

---

## Phase 2 — Client APIs (library beyond search)

Build on authenticated client (port shapes from Python `ytmusicapi` as needed):

- [x] `GetLibraryPlaylists`
- [x] `GetPlaylist(id)` (+ pagination)
- [x] `GetLikedSongs` / library songs
- [x] `GetHistory` (same trust model as liked songs — local agent)
- [x] `Search` hardened with auth (better personalization / Music catalog)
- [x] `GetWatchPlaylist` / radio seed from `videoId` (radio / continuum)
- [x] Return stable IDs an AI agent can cast: `videoId`, optional `playlistId`, title, artists, duration
- [x] Unit tests with recorded fixtures (no live cookies in CI)

**Out of scope for v1:** playlist mutate (create/add/delete), likes write — add later if needed.

---

## Phase 3 — MCP server (stdio)

Thin MCP over the client — keep the AI agent tool surface small.

### v1 tools

- [x] `search_tracks` — query → list of tracks (`videoId`, title, artists, …)
- [x] `get_library_playlists`
- [x] `get_playlist` — playlist id → tracks
- [x] `get_liked_songs` (limit)
- [x] `get_history` (limit)
- [x] `get_watch_playlist` / radio from a seed `videoId`
- [x] `get_track` / `get_lyrics` (lyrics when YTM exposes them)

### server plumbing

- [x] stdio MCP via `github.com/modelcontextprotocol/go-sdk`
- [x] `--version` / `--self-test` (auth present? search smoke?)
- [x] Structured errors an AI agent can act on
- [x] Logging to stderr only (never stdout — stdio protocol)

---

## Phase 4 — Playback bridge (don’t strand the agent on royalty-free MP3s)

Search alone is not enough. Nest / Cast needs a path that understands YouTube.

- [x] Document contract: this MCP returns `videoId` + `https://music.youtube.com/watch?v=…` / `youtube.com/watch?v=…`
- [ ] Coordinate with a Cast MCP (e.g. mcp-beam / go2tv): **cast by video ID** (YouTube receiver), not only raw media URLs
- [ ] Until Cast supports video-ID: temporary guidance for the AI agent (don’t invent free-MP3 fallbacks)
- [x] Optional helper tool: `format_cast_target(videoId)` → payload Cast expects

---

## Phase 5 — Wire into an AI agent host

- [x] Dockerfile stage in `docker_open_claw`: fetch release → `/usr/local/bin/youtube-go-mcp`
- [x] MCP server / bundle config + grant on the main agent (`ytmusic`)
- [x] Compose: mount `secrets/ytmusic/`, `YTMUSIC_HEADERS_PATH`
- [x] Host docs: `docker_open_claw/docs/ytmusic.md` + `make ytmusic-auth`
- [x] Agent tools doc: search → pick track → Cast to a room / device
- [x] Pin via `YOUTUBE_GO_MCP_VERSION` (default `v0.0.1`)

---

## Phase 6 — Hardening

- [x] Rate-limit / backoff on InnerTube 429s
- [x] Client version config (env override without rebuild)
- [x] Cookie refresh docs when Premium session dies
- [x] Release binaries (GoReleaser on `v*` tags)

---

## Decisions / notes

| Topic | Decision |
|---|---|
| Language | Go only (static binary for distroless agent hosts) |
| Auth | Browser cookies / headers — not YouTube Data API v3 for library |
| Official Data API | Skip for Music library; may use later only if needed for something else |
| Cast | Separate MCP (e.g. mcp-beam); this repo sources identity + IDs |
| Python rewrite | No — expand this client instead |

---

## Immediate next (when we work here)

1. ~~Module rename + package layout (`cmd/` + `internal/ytmusic`)~~
2. ~~Browser auth + one authenticated smoke (`GetLibraryPlaylists`)~~
3. ~~Minimal MCP: `search_tracks` + `get_library_playlists` over stdio~~
4. ~~Wire into Tim (`docker_open_claw`)~~ — Cast video-ID handoff still open
5. ~~`GetPlaylist` / `GetLikedSongs` + matching MCP tools~~

---

## Smoke checklist (later)

```bash
go test ./...
go build -o bin/youtube-go-mcp ./cmd/youtube-go-mcp
./bin/youtube-go-mcp --self-test
# With headers.json mounted:
# ask the AI agent: "search YouTube Music for …" / "list my Music playlists"
```
