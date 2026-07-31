# Cast contract — source IDs, play elsewhere

This MCP **sources** YouTube identity (`videoId`). It does **not** cast to
Chromecast / Nest / TVs. Playback is pluggable:

- a Cast MCP (e.g. mcp-beam → `beam_youtube_video` / host `cast__youtube_beam_video`)
- Google Assistant / Home routines
- the YouTube app “Cast” button
- anything else that accepts a YouTube video id

## Cast-ready fields

Every video row from `videos_search`, `videos_get`, `playlists_get`, and
`library_list_liked_videos` includes:

| Field | Always | Purpose |
|---|---|---|
| `videoId` | yes | Canonical id |
| `video_id` | yes | Same id (snake_case for bridges that expect it) |
| `url` | yes | `https://www.youtube.com/watch?v=…` |
| `musicUrl` | **only if music-like** | `https://music.youtube.com/watch?v=…` when `categoryId=10` or channel ends with `- Topic` |
| `preferredMediaKind` | **only if music-like** | `"audio"` — Nest Mini–friendly hint for the cast bridge |

`musicUrl` is a convenience link — **never required for playback**. Cast by
`video_id` / `videoId` on a YouTube receiver.

`cast_format_target` returns handoff fields plus bridge hints:

| Field | Purpose |
|---|---|
| `videoId` / `video_id` / `url` | Same as video rows |
| `preferredMediaKind` | `"audio"` or `"video"` |
| `preferredContentType` | `audio/mp4` when audio-preferring |
| `castMetadataType` | `3` (music track) or `0` (generic) |
| `title` / `artist` | Optional Cast metadata (from args or Data API when auth ready) |
| `castHint` | Human/agent guidance for mcp-beam |

Set `audioOnlyTarget: true` when `devices_list.is_audio_only` (Nest Mini).
`musicLikely: true` (or a music-leaning Data API row) also prefers audio.

## Agent flows (same handoff)

**Play a song**

1. `youtube__videos_search` with `musicOnly: true` (or liked videos)
2. Pick a result’s `video_id`
3. Optional: `youtube__cast_format_target`
4. Call your Cast bridge with that `video_id`

**Play a video (talk, clip, essay)**

1. `youtube__videos_search` (no `musicOnly`) or `youtube__videos_get`
2. Same steps 2–4 — **identical** `video_id` handoff

There is no separate “music cast” path. Music and video share one YouTube
receiver contract.

## Nest Mini / audio-only targets

Nest Hub Max (display) usually tolerates the YouTube Cast receiver and even
`video/*` containers. **Google Nest Mini** (and other headless Cast speakers)
often reject or silently drop loads when:

- `contentType` is `video/*`, or the container has a video track
- the YouTube MDX lounge path leaves `_audioOnly=false`
- LOAD metadata is typed as generic video instead of a music track

This MCP still only sources `videoId`. The playback bridge (e.g. mcp-beam) must
adapt per device:

| Path | Audio-only device expectation |
|---|---|
| `youtube_beam_video` | YouTube receiver + lounge `_audioOnly=true` when `is_audio_only` |
| `media_beam` (direct URL) | Demuxed audio URL, `contentType: audio/mp4` (or `audio/webm`), `streamType: BUFFERED`, music metadata `type: 3` |

Direct `audio/mp4` also works on display Nest/Hub devices, so preferring audio
for music is the resilient default. Do **not** expect this MCP to return itag
140 / M4A stream URLs — Data API v3 does not provide them.

## Anti-patterns

- Do not invent royalty-free MP3 / direct stream URLs from this MCP
- Do not pass `url` / `musicUrl` to a generic “play media URL” tool unless that
  tool explicitly understands YouTube watch URLs
- Prefer `video_id` on a YouTube-aware Cast tool
- Do not assume Nest Mini can play the same `video/*` LOAD as Nest Hub Max
