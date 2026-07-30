# Auth (OAuth preferred, browser cookies fallback)

Library tools (`library_list_playlists`, `library_list_liked_songs`, private
playlists via `playlists_get`) need a signed-in **YouTube Music** identity —
the same Google account that owns your Liked Songs / Library on
[music.youtube.com](https://music.youtube.com).

**Prefer OAuth.** Browser cookie jars die when you log into YouTube again in the
same profile. OAuth uses a refreshable Bearer token that survives normal browsing.

If both `YTMUSIC_OAUTH_PATH` and `YTMUSIC_HEADERS_PATH` are set, **OAuth wins**.

---

## Option A — OAuth (recommended)

Same TV/device flow as [ytmusicapi](https://ytmusicapi.readthedocs.io/en/stable/setup/oauth.html).

### One-time Google Cloud setup

1. [Google Cloud Console](https://console.cloud.google.com/) → create/select a project.
2. Enable **YouTube Data API v3**.
3. **APIs & Services → Credentials → Create credentials → OAuth client ID**.
4. Application type: **TVs and Limited Input devices** (not “Desktop” / “Web”).
5. Copy the **client id** and **client secret**.

The GCP project owner email does **not** choose which YouTube account the token
uses. The project only hosts the OAuth client. Identity is decided when you
approve the device code in the browser (next step).

### Mint `oauth.json`

```bash
export YTMUSIC_OAUTH_CLIENT_ID='….apps.googleusercontent.com'
export YTMUSIC_OAUTH_CLIENT_SECRET='…'
./bin/youtube-go-mcp auth oauth --out oauth.json
# Open the printed URL, enter the code, approve access.
./bin/youtube-go-mcp auth oauth --validate oauth.json
```

**Critical — approve as the right Google account**

1. Prefer an **incognito / clean browser profile** with only your personal Gmail
   signed in (the one that owns music.youtube.com Library / Liked Songs).
2. Open the printed `google.com/device` link, enter the code, and confirm the
   account picker shows **that** account — not work, not a Brand Account, not
   a secondary profile.
3. If Chrome has several Google accounts, it is very easy to mint a valid token
   for the wrong face. Auth “works”; Liked Songs look empty or foreign.

### Confirm who the token is

After minting (with the same env vars you’ll use at runtime):

```bash
export YTMUSIC_OAUTH_PATH=$PWD/oauth.json
export YTMUSIC_OAUTH_CLIENT_ID='…'
export YTMUSIC_OAUTH_CLIENT_SECRET='…'
./bin/youtube-go-mcp auth oauth --whoami
./bin/youtube-go-mcp auth oauth --probe-library
```

`--whoami` prints Google tokeninfo plus your YouTube **channel title/id**
(`youtube` scope often omits email from tokeninfo — trust the channel).

`--probe-library` reports whether Liked Songs returned a shelf / track count.
If `likedTracksParsed` is 0, read the `hint` field.

### Empty Liked Songs after a “perfect” mint

Common causes beyond wrong Gmail:

1. **Missing visitor id** — this client now fetches `X-Goog-Visitor-Id` like
   ytmusicapi; rebuild/redeploy if you’re on an older binary.
2. **Brand Account** — Music library lives on a Brand Account while OAuth bound
   the personal Google login. Set `YTMUSIC_ON_BEHALF_OF_USER` to the Brand
   Account id from
   [myaccount.google.com/brandaccounts](https://myaccount.google.com/brandaccounts)
   (the numeric id in the URL after `/b/`).
3. **Parser vs identity** — `--probe-library` distinguishes “no shelf in JSON”
   from “shelf present, zero tracks.”

### Run the MCP

```bash
export YTMUSIC_OAUTH_PATH=$PWD/oauth.json
export YTMUSIC_OAUTH_CLIENT_ID='….apps.googleusercontent.com'
export YTMUSIC_OAUTH_CLIENT_SECRET='…'
./bin/youtube-go-mcp --self-test
```

`--self-test` should list library playlists / liked / history when auth is
correct. Empty library with a passing token usually means **wrong Google
identity**, not a bad GCP project.

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

**Ops tip:** mint cookies from a dedicated browser profile you never use for
daily YouTube. Prefer switching to OAuth when you can.

### When the cookie session dies

Typical causes: sign-out, “log back in”, cleared cookies, revoked device, bot checks.

Symptoms: `session expired` / HTTP 401–403 on library tools while search still works.

Refresh: re-export headers into the same path (MCP reloads on mtime), or switch to OAuth.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Library / liked empty, search works, no HTTP error | Token minted under the **wrong Google account** (or Brand Account) | `tokeninfo` check → re-mint in incognito as personal Gmail |
| `401` / `403` / `session expired` on library | Revoked/expired token, wrong client id/secret for refresh, or dead cookies | Re-run `auth oauth` (keep same client id/secret) or re-export headers |
| Mint / refresh fails with `invalid_client` | Client type isn’t **TVs and Limited Input devices**, or secret mismatch | Recreate TV OAuth client; enable YouTube Data API v3 |
| “I use the same email for GCP…” but library is wrong | GCP project email ≠ account that clicked **Allow** on `google.com/device` | Ignore GCP owner; fix device-approval account |
| Cookies work, OAuth doesn’t (or vice versa) | Both env paths set → **OAuth wins**; or different identities | Unset the unused path; compare `tokeninfo` email vs browser session |
| OAuth app in “Testing” | Only listed test users can authorize | Add your Gmail as a test user, or publish the app |

---

## Env reference

| Env | Purpose |
|---|---|
| `YTMUSIC_OAUTH_PATH` | Path to `oauth.json` (**preferred**) |
| `YTMUSIC_OAUTH_CLIENT_ID` | Google OAuth client id |
| `YTMUSIC_OAUTH_CLIENT_SECRET` | Google OAuth client secret |
| `YTMUSIC_ON_BEHALF_OF_USER` | Optional Brand Account id for library requests |
| `YTMUSIC_HEADERS_PATH` | Path to browser `headers.json` (legacy) |
| `YTMUSIC_CLIENT_VERSION` | Override InnerTube `clientVersion` |
| `YTMUSIC_MIN_REQUEST_INTERVAL_MS` | Min spacing between InnerTube calls (default `250`) |
| `YTMUSIC_MAX_RETRIES` | Retries after HTTP 429/503 (default `3`) |
