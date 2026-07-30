# Auth (OAuth preferred, browser cookies fallback)

Library tools (`library_list_playlists`, `library_list_liked_songs`, private
playlists via `playlists_get`) need a signed-in YouTube Music identity.

**Prefer OAuth.** Browser cookie jars die when you log into YouTube again in the
same profile. OAuth uses a refreshable Bearer token that survives normal browsing.

---

## Option A — OAuth (recommended)

Same TV/device flow as [ytmusicapi](https://ytmusicapi.readthedocs.io/en/stable/setup/oauth.html).

### One-time Google Cloud setup

1. [Google Cloud Console](https://console.cloud.google.com/) → create/select a project.
2. Enable **YouTube Data API v3**.
3. **APIs & Services → Credentials → Create credentials → OAuth client ID**.
4. Application type: **TVs and Limited Input devices**.
5. Copy the **client id** and **client secret**.

### Mint `oauth.json`

```bash
export YTMUSIC_OAUTH_CLIENT_ID='….apps.googleusercontent.com'
export YTMUSIC_OAUTH_CLIENT_SECRET='…'
./bin/youtube-go-mcp auth oauth --out oauth.json
# Open the printed URL, enter the code, approve access.
./bin/youtube-go-mcp auth oauth --validate oauth.json
```

### Run the MCP

```bash
export YTMUSIC_OAUTH_PATH=$PWD/oauth.json
export YTMUSIC_OAUTH_CLIENT_ID='….apps.googleusercontent.com'
export YTMUSIC_OAUTH_CLIENT_SECRET='…'
./bin/youtube-go-mcp --self-test
```

The process refreshes `access_token` automatically when it is near expiry and
rewrites `oauth.json` in place. Client id/secret must stay available at runtime
(they are required for refresh).

**Never commit `oauth.json` or client secrets.** Mount them as host secrets
(e.g. `secrets/ytmusic/oauth.json`).

---

## Option B — Browser cookies (legacy)

Works, but fragile: logging into YouTube in the browser profile that minted the
cookies often invalidates the jar within minutes/hours.

### Setup

1. Open [music.youtube.com](https://music.youtube.com) and sign in (Premium recommended).
2. DevTools (**F12**) → **Network** → filter `browse`.
3. Open **Library** so a `POST` to `/youtubei/v1/browse` appears.
4. Copy **`cookie`** (must include `__Secure-3PAPISID` or `SAPISID`) and **`x-goog-authuser`**.
5. Export:

```bash
./bin/youtube-go-mcp auth browser --out headers.json
export YTMUSIC_HEADERS_PATH=$PWD/headers.json
./bin/youtube-go-mcp auth browser --validate "$YTMUSIC_HEADERS_PATH"
./bin/youtube-go-mcp --self-test
```

If both OAuth and browser paths are configured, **OAuth wins**.

### When the cookie session dies

Typical causes: sign-out, “log back in”, cleared cookies, revoked device, bot checks.

Symptoms: `session expired` / HTTP 401–403 on library tools while search still works.

Refresh: re-export headers into the same path (MCP reloads on mtime), or switch to OAuth.

**Ops tip:** mint cookies from a dedicated browser profile you never use for daily YouTube.

---

## Env reference

| Env | Purpose |
|---|---|
| `YTMUSIC_OAUTH_PATH` | Path to `oauth.json` (**preferred**) |
| `YTMUSIC_OAUTH_CLIENT_ID` | Google OAuth client id |
| `YTMUSIC_OAUTH_CLIENT_SECRET` | Google OAuth client secret |
| `YTMUSIC_HEADERS_PATH` | Path to browser `headers.json` (legacy) |
| `YTMUSIC_CLIENT_VERSION` | Override InnerTube `clientVersion` |
| `YTMUSIC_MIN_REQUEST_INTERVAL_MS` | Min spacing between InnerTube calls (default `250`) |
| `YTMUSIC_MAX_RETRIES` | Retries after HTTP 429/503 (default `3`) |
