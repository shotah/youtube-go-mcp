# youtube-go-mcp — YouTube MCP pivot

Working board (fresh). Prior Music/InnerTube phases are **done / closed**; this doc maps the move to a real **YouTube Data API v3** MCP.

**Why:** InnerTube (`youtubei/v1`) rejects durable OAuth Bearer for Music library; cookie replay is fragile for always-on agents. Official Data API v3 + refreshable OAuth is the long-lived path. Same `videoId` still casts via whatever player bridge you use (mcp-beam, another Cast sender, Assistant, manual) — now **any** YouTube video, not only Music catalog.

**Music stays in scope as a use case.** We ditch InnerTube, not music. Anything we can honestly pull from Data API v3 for taste / music discovery, we explore and keep. Prefer `youtube.com/watch?v=…`; optional `musicUrl` as a convenience link when content looks like music.

---

## North star

| Pillar | Decision |
|---|---|
| Product | **YouTube MCP** (search, library-ish reads, metadata) for AI agents |
| Backend | YouTube **Data API v3** only — **ditch InnerTube v1** entirely |
| Auth | OAuth device/TV flow + **refresh token** (long-lived). Drop browser cookies / SAPISIDHASH |
| Cast | Return castable `videoId` (+ `video_id` / `url`); playback bridge is pluggable (not required to be mcp-beam) |
| Music | **Keep an eye on** — extract what v3 allows (likes, category/Topic filters, music-leaning search); never reintroduce InnerTube for it |
| Tool naming | **Hard requirement** — follow shared contract (link below). Qwen closest-match must not guess. |
| Binary / module name | Keep `youtube-go-mcp` for now; rebrand docs/tools language to YouTube |

---

## MCP naming (locked — do not freestyle)

**Canonical doc:** [ai-gantry `docs/mcp-naming.md`](https://github.com/shotah/ai-gantry/blob/main/docs/mcp-naming.md)

Hosts expose `{server_id}__{tool}`. Server id in ai-gantry stays **`youtube`**.

| Rule | This package |
| --- | --- |
| `{service}_{verb}_{object…}` | Yes |
| Never `youtube_` on the tool | Host already makes `youtube__…` |
| Stable verbs | `search`, `list`, `get`, `format` |
| Data API era noun | **`videos_`** (not Music/InnerTube `tracks_`) |
| Shared nouns | `playlists_`, `library_`, `cast_` (handoff only; playback stays mcp-beam) |
| Descriptions | Lead with agent intent (“Search YouTube videos…”) |
| Tests | No server-id prefix; assert registered names |

**Locked host forms after Phase C:**

| Tool | Host call |
| --- | --- |
| `videos_search` | `youtube__videos_search` |
| `videos_get` | `youtube__videos_get` |
| `playlists_get` | `youtube__playlists_get` |
| `library_list_playlists` | `youtube__library_list_playlists` |
| `library_list_liked_videos` | `youtube__library_list_liked_videos` |
| `cast_format_target` | `youtube__cast_format_target` |

Cast playback remains **`cast__youtube_beam_video`** / `cast__devices_list` on
server `cast` — do not re-home beam tools into this binary.

---

## Closed (do not reopen)

- [x] Music InnerTube client + MCP v1 tools (search, library, likes, history, radio, lyrics)
- [x] Browser cookie auth + OAuth device flow plumbing
- [x] Visitor-id / probe / Brand Account investigation — OAuth + InnerTube library is a dead end
- [x] Host wiring, CI, coverage gate, releases through `v0.2.2`

---

## Phase A — Product + auth contract

- [x] Rewrite mental model in README / `docs/auth.md`: YouTube Data API v3 + OAuth refresh; InnerTube/cookies deprecated
- [x] Env naming: prefer `YOUTUBE_OAUTH_*`, legacy `YTMUSIC_OAUTH_*` aliases still work
- [x] Scopes: mint `youtube`; document `youtube.readonly` as read-only remint option
- [x] Quota story: daily quota, pagination, backoff on 403 rateLimitExceeded (docs/auth.md)
- [x] Self-test / `--probe-data-api`: OAuth refresh + `channels.list?mine=true` + `videos.list?myRating=like`

---

## Phase B — Data API client (replace InnerTube)

Package: `internal/youtube` (OAuth via `TokenSource`; `ytmusic.OAuthSession.BearerToken` implements it).

- [x] HTTP client: `googleapis.com/youtube/v3/*` + Bearer from existing OAuth refresh
- [x] `ChannelMine` — identity (title, id, `relatedPlaylists.likes` / uploads)
- [x] `SearchVideos` — `q`, `type=video`, optional `MusicOnly` → `videoCategoryId=10`
- [x] `GetVideo` / `GetVideos` / `ListLikedVideos` — by id + `myRating=like`
- [x] `ListMyPlaylists` / `ListPlaylistItems` — mine + by playlist id
- [x] Agent-friendly structs: `videoId`, title, channel, duration, `url`, optional `musicUrl`
- [x] Unit tests with httptest fixtures (no live token in CI)
- [x] `ProbeDataAPI` delegates to `internal/youtube`
- [x] **Hard delete** InnerTube + browser cookie auth (`ytmusic` is OAuth/WhoAmI/probe only)

---

## Phase C — MCP tool surface (YouTube-shaped)

Follow [mcp-naming.md](https://github.com/shotah/ai-gantry/blob/main/docs/mcp-naming.md).
**Decision locked:** `tracks_*` → `videos_*` (Data API honesty + matching).

### Keep / reshape (old Music → new)

| Old (InnerTube / Music) | New tool | Host after | Direction |
|---|---|---|---|
| `tracks_search` | `videos_search` | `youtube__videos_search` | Data API `search.list`; optional `musicOnly` / `videoCategoryId=10` |
| `tracks_get` | `videos_get` | `youtube__videos_get` | `videos.list` by id |
| `playlists_get` | `playlists_get` | `youtube__playlists_get` | `playlistItems` by `PL…` |
| `library_list_playlists` | `library_list_playlists` | `youtube__library_list_playlists` | `playlists.list?mine=true` |
| `library_list_liked_songs` | `library_list_liked_videos` | `youtube__library_list_liked_videos` | `videos.list?myRating=like` (or `LL`) |
| `cast_format_target` | `cast_format_target` | `youtube__cast_format_target` | Any `videoId`; player-agnostic hint |

Checklist:

- [x] Register only the **new** names (no dual aliases)
- [x] Tool descriptions lead with intent (see mcp-naming.md)
- [x] Name assertion tests (no `youtube_` tool prefix)
- [ ] ai-gantry `TOOLS.md` / docs: `youtube__tracks_*` → `youtube__videos_*` in same consumer change (**host repo**)

### Dropped (InnerTube-only)

| Old tool | Why |
|---|---|
| `library_list_history` | Watch/Music history not in Data API |
| `tracks_list_watch_playlist` | InnerTube radio |
| `tracks_get_lyrics` | Music InnerTube lyrics |

### Agent flows to support

1. Search (optionally music-leaning) → pick `videoId` → cast via chosen bridge  
2. List liked videos → filter/suggest music-ish → cast  
3. List my playlists → get items → cast  
4. Resolve metadata for any `videoId` (essay, MV, clip — all fair game)

---

## Phase D — Cast (pluggable) + music-friendly output

- [x] Docs: this MCP **sources** IDs; casting is separate — [docs/cast.md](docs/cast.md) + README
- [x] Outputs always include cast-ready fields (`videoId`, `video_id`, `url`)
- [x] Optional `musicUrl` when content looks like music — never required for playback (`cast_format_target` omits it)
- [x] Host/agent examples: “play this video” and “play this song” share the same `videoId` handoff

---

## Phase E — Music via Data API (keep an eye on)

Not a second backend — thin helpers on top of v3 ([docs/music.md](docs/music.md)):

- [x] Taste signal via `--probe-data-api` (`likedVideos` + `likedMusicCategoryCount`) — live account varies
- [x] Music-leaning heuristics: category 10, `… - Topic`, Official Audio / Provided to YouTube / lyrics / visualizer hints
- [x] Enrich rows: `categoryId`, `channelTitle`, `durationSec`, `musicLikely`, optional `musicUrl`
- [x] `musicOnly=true` on `videos_search` and `library_list_liked_videos`

---

## Phase F — Cutover + release

- [x] Default binary: Data API v3 only (InnerTube gone)
- [x] `--self-test` / `auth oauth --whoami` / `--probe-data-api`
- [x] README: YouTube · MCP · Cast; music as use case on v3
- [ ] Semver note for breaking tool renames (`v0.3.0` or `v1.0.0`)
- [ ] Host updates: ai-gantry `TOOLS.md` + OAuth-only env (`youtube__videos_*`)
- [ ] Smoke: cast a music video **and** a non-music video through whatever player you use

---

## Phase G — Supporting Google Nest Mini

**Status:** this package done (hints + docs); playback fix tracked in mcp-beam  

**Why:** Nest Hub Max (display) accepts YouTube / video containers; Nest Mini
(audio-only Cast) rejects or silently drops loads when MIME/container/app id
assume a video receiver. No second MCP server — fix the handoff between this
package (source) and mcp-beam (playback).

### Root cause (Cast path)

| Factor | Display (Hub Max) | Audio-only (Nest Mini) |
|---|---|---|
| MIME / `contentType` | Tolerates `video/*`, HLS with video | Rejects `video/*` or containers with video tracks it cannot decode |
| Stream shape | Combined A/V OK | Needs demuxed audio (M4A/AAC/Opus → `audio/mp4` / `audio/webm`) **or** a YouTube receiver path that declares audio-only |
| Receiver / metadata | Generic / video metadata OK | Prefer Default Media Receiver + `streamType: BUFFERED` + `audio/*`, or YouTube MDX with `_audioOnly=true`; music metadata `type: 3` (MusicTrackMediaMetadata) |

Native YouTube Cast app (`233637DE`) on speakers is fragile unless lounge
params advertise audio-only. Direct `media_beam` of a video container fails
harder. **Audio/`audio/mp4` payloads work on both Hub Max and Nest Mini.**

### This package (`youtube-go-mcp`)

Data API v3 still does **not** return playable stream URLs. Nest Mini support
here is contract + optional enrichment — not a player.

- [x] Docs (`docs/cast.md`): Nest Mini / audio-only target notes; bridge must
      send `audio/*` (or YouTube MDX `_audioOnly`) — do not invent MP3 URLs
- [x] `cast_format_target` (and video rows as needed): optional hints for
      audio-capable bridges — `preferredMediaKind`, `preferredContentType`,
      `castMetadataType`, `title`/`artist`, `audioOnlyTarget` input
- [x] Decision lock: **stream extraction stays out of this binary** unless we
      explicitly reopen player/yt-dlp (conflicts with Phase B Data-API-only).
      Prefer mcp-beam (or a dedicated extractor) to resolve demuxed audio when
      `devices_list.is_audio_only` is true
- [ ] Smoke checklist: Hub Max + Nest Mini with same `video_id` handoff via
      mcp-beam after bridge fix

### Partner package (`mcp-beam`) — track there

See mcp-beam `TODO.md` → **Supporting Google Nest Mini**.

### Architecture (single cast endpoint)

```text
youtube-go-mcp          mcp-beam
  videoId / hints  -->  devices_list (is_audio_only?)
                        ├─ display  → YouTube receiver / video OK
                        └─ speaker  → _audioOnly or audio/* LOAD
```

---

## Immediate next

1. Phase G: Nest Mini — doc + cast hints here; mcp-beam `_audioOnly` / LOAD MIME  
2. Update ai-gantry `TOOLS.md` / host mcp config for `youtube__videos_*` (breaking rename)  
3. Cut release when host is ready  

---

## Open questions

1. Rename binary/module eventually (`youtube-mcp`) or keep `youtube-go-mcp`?  
2. ~~`tracks_*` vs `videos_*`~~ — **locked: `videos_*`** (mcp-naming.md + table above).  
3. Default search: all of YouTube, or music-leaning with opt-out?  
4. Write tools later (`videos.rate`, playlist mutate) — after read path is solid?  
5. ~~Shared naming doc?~~ — **yes:** [ai-gantry/docs/mcp-naming.md](https://github.com/shotah/ai-gantry/blob/main/docs/mcp-naming.md)  
6. Nest Mini: keep YouTube MDX + `_audioOnly`, or demuxed `audio/mp4` via Default Media Receiver (or both)?  

