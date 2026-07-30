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

`musicUrl` is a convenience link — **never required for playback**. Cast by
`video_id` / `videoId` on a YouTube receiver.

`cast_format_target` returns `{ videoId, video_id, url, castHint }` when you
already have an id and only need the handoff shape.

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

## Anti-patterns

- Do not invent royalty-free MP3 / direct stream URLs from this MCP
- Do not pass `url` / `musicUrl` to a generic “play media URL” tool unless that
  tool explicitly understands YouTube watch URLs
- Prefer `video_id` on a YouTube-aware Cast tool
