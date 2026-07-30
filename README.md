<p align="center">
  <img src="docs/assets/banner.png" alt="youtube-go-mcp — YouTube · MCP · Cast" width="100%">
</p>

# youtube-go-mcp

<p align="center">
  <a href="https://github.com/shotah/youtube-go-mcp/actions/workflows/ci.yml"><img src="https://github.com/shotah/youtube-go-mcp/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/shotah/youtube-go-mcp/actions/workflows/release.yml"><img src="https://github.com/shotah/youtube-go-mcp/actions/workflows/release.yml/badge.svg" alt="Release"></a>
  <a href="https://github.com/shotah/youtube-go-mcp/actions/workflows/ci.yml"><img src="https://github.com/shotah/youtube-go-mcp/raw/gh-pages/badges/coverage.svg" alt="Coverage"></a>
  <a href="https://pkg.go.dev/github.com/shotah/youtube-go-mcp"><img src="https://pkg.go.dev/badge/github.com/shotah/youtube-go-mcp.svg" alt="Go Reference"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/github.com/shotah/youtube-go-mcp" alt="Go version">
  <a href="LICENSE"><img src="https://img.shields.io/github/license/shotah/youtube-go-mcp" alt="License"></a>
</p>

Static Go [MCP](https://modelcontextprotocol.io) server for **YouTube Data API v3** — search, playlists, liked videos, and cast-ready `videoId`s. Long-lived auth via OAuth refresh tokens. Playback is separate (Cast bridge, Assistant, etc.).

Tool naming follows [ai-gantry mcp-naming](https://github.com/shotah/ai-gantry/blob/main/docs/mcp-naming.md). Server id `youtube` → host calls like `youtube__videos_search`.

## Tools

| Tool | Auth | Description |
|---|---|---|
| `videos_search` | required | Search videos (`musicOnly` → category 10) |
| `videos_get` | required | Metadata for a `videoId` |
| `playlists_get` | required | Videos in a playlist (`PL…` / `LL…`) |
| `library_list_playlists` | required | Channel-owned playlists |
| `library_list_liked_videos` | required | Thumbs-up likes (`musicOnly` keeps music-leaning) |
| `cast_format_target` | no | `videoId` / `video_id` + URLs for a playback bridge |

Music is a **filter on Data API**, not a second backend — see [docs/music.md](docs/music.md).

## Build

```bash
make help
make check
make cli
make self-test   # needs OAuth env
```

## Auth

```bash
export YOUTUBE_OAUTH_CLIENT_ID='….apps.googleusercontent.com'
export YOUTUBE_OAUTH_CLIENT_SECRET='…'
./bin/youtube-go-mcp auth oauth --out oauth.json
export YOUTUBE_OAUTH_PATH=$PWD/oauth.json
./bin/youtube-go-mcp auth oauth --whoami
./bin/youtube-go-mcp auth oauth --probe-data-api
./bin/youtube-go-mcp --self-test
```

Legacy `YTMUSIC_OAUTH_*` env names still work. Full detail: [docs/auth.md](docs/auth.md).

**Never commit `oauth.json` / client secrets.**

## Run as MCP (stdio)

```bash
YOUTUBE_OAUTH_PATH=/path/to/oauth.json \
YOUTUBE_OAUTH_CLIENT_ID=… \
YOUTUBE_OAUTH_CLIENT_SECRET=… \
./bin/youtube-go-mcp
```

```json
{
  "mcpServers": {
    "youtube": {
      "command": "/usr/local/bin/youtube-go-mcp",
      "env": {
        "YOUTUBE_OAUTH_PATH": "/secrets/youtube/oauth.json",
        "YOUTUBE_OAUTH_CLIENT_ID": "….apps.googleusercontent.com",
        "YOUTUBE_OAUTH_CLIENT_SECRET": "…"
      }
    }
  }
}
```

## Cast contract

This MCP **sources** `videoId`s — it does not talk to Chromecast/Nest itself.
Playback is whatever bridge you use (Cast MCP, Assistant, manual).

| Field | When |
|---|---|
| `videoId` / `video_id` / `url` | Always on video rows |
| `musicUrl` | Only when content looks like music (category 10 / Topic channel) |

**Same handoff for song or video:** take `video_id` → pass to your YouTube Cast
bridge (e.g. mcp-beam / `cast__youtube_beam_video`). Details: [docs/cast.md](docs/cast.md).

## License

MIT (see [LICENSE](LICENSE)).
