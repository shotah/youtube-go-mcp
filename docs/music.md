# Music via Data API (thin filter)

There is **no** official YouTube Music Liked Songs / listen-history API here.
We keep a **thin music lens** on Data API v3 so agents can still taste-match
and cast songs when the signal exists.

## What works

| Signal | How |
|---|---|
| Music search | `videos_search` with `musicOnly: true` → `videoCategoryId=10` |
| Liked taste | `library_list_liked_videos` (YouTube thumbs-up). Optional `musicOnly: true` |
| Heuristics | `categoryId=10`, channel `… - Topic`, title/description hints (“Official Audio”, “Provided to YouTube by”, …) |
| Enrichment | Rows include `categoryId`, `channelTitle`, `durationSec`, `musicLikely`, optional `musicUrl` |

Check your account with:

```bash
./bin/youtube-go-mcp auth oauth --probe-data-api
```

`likedMusicCategoryCount` on that probe is how many of the first liked page are
category 10. If it’s 0 but you have Music likes in the Music app, those lists
diverged — still use `musicOnly` search + whatever thumbs-up music you have.

## What does **not** work (honest gaps)

- YouTube Music **Liked Songs** shelf (`LM`)
- Music listening **history**
- Music-only radio / continuum from InnerTube

## Agent tips

```text
# Find a song
youtube__videos_search  query query="…", musicOnly=true

# Taste from likes (music-leaning)
youtube__library_list_liked_videos  musicOnly=true

# Cast — same as any video
youtube__cast_format_target  videoId=<id>
→ pass video_id to your Cast bridge
```

Crossed fingers: if your thumbs-up history overlaps Music taste, this layer is
enough for a useful agent. If not, search + `musicOnly` still covers discovery.
