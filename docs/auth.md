# Auth — YouTube Data API v3 + OAuth

Durable auth is OAuth with a **refresh token**. Access tokens expire ~1 hour;
the process refreshes them automatically.

Browser cookies / InnerTube auth have been **removed**.

---

## Env names

| Preferred | Legacy alias (still works) |
|---|---|
| `YOUTUBE_OAUTH_PATH` | `YTMUSIC_OAUTH_PATH` |
| `YOUTUBE_OAUTH_CLIENT_ID` | `YTMUSIC_OAUTH_CLIENT_ID` |
| `YOUTUBE_OAUTH_CLIENT_SECRET` | `YTMUSIC_OAUTH_CLIENT_SECRET` |

If both preferred and legacy are set, **preferred wins**.

---

## Scopes

Device flow mints `https://www.googleapis.com/auth/youtube`.
`youtube.readonly` also works for reads if you remint with that scope.

---

## Mint `oauth.json`

```bash
export YOUTUBE_OAUTH_CLIENT_ID='….apps.googleusercontent.com'
export YOUTUBE_OAUTH_CLIENT_SECRET='…'
./bin/youtube-go-mcp auth oauth --out oauth.json
./bin/youtube-go-mcp auth oauth --validate oauth.json
export YOUTUBE_OAUTH_PATH=$PWD/oauth.json
./bin/youtube-go-mcp auth oauth --whoami
./bin/youtube-go-mcp auth oauth --probe-data-api
./bin/youtube-go-mcp --self-test
```

GCP setup: enable **YouTube Data API v3**, OAuth client type **TVs and Limited Input devices**.

---

## Quota

Projects have a daily YouTube Data API quota. Prefer pagination sparingly on
`search.list`. HTTP **403** with `quotaExceeded` / `rateLimitExceeded` means
back off until reset (or request a higher quota).

---

## Music via Data API

Official API does **not** expose YouTube Music Liked Songs / listen history.
Use `library_list_liked_videos` + `videos_search` with `musicOnly=true` and
heuristic flags (`musicLikely` / `musicUrl`). Details: [music.md](music.md).

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `invalid_client` | Recreate **TVs and Limited Input devices** client |
| Empty channel on `--whoami` | Remint as the correct Google account |
| `likedVideos: 0` | No YouTube thumbs-up likes (Music likes ≠ this list) |
| Quota / 403 | Wait for daily reset; reduce search volume |

**Never commit `oauth.json` or client secrets.**
