# Browser auth & cookie refresh

YouTube Music library tools (`get_library_playlists`, `get_liked_songs`, private playlists) use your **browser session**, not a Google API key.

## One-time setup

1. Open [music.youtube.com](https://music.youtube.com) and sign in (Premium session recommended).
2. DevTools → **Network** → filter `browse`.
3. Open **Library** (or scroll) so a `POST` to `/youtubei/v1/browse` appears.
4. Right-click the request → **Copy** → **Copy request headers**.
5. Export headers:

```bash
./bin/youtube-go-mcp auth --out headers.json
# paste headers, then EOF (Ctrl-Z Enter on Windows / Ctrl-D on Unix)
export YTMUSIC_HEADERS_PATH=$PWD/headers.json
./bin/youtube-go-mcp auth --validate "$YTMUSIC_HEADERS_PATH"
./bin/youtube-go-mcp --self-test
```

Required fields: `cookie` (must include `__Secure-3PAPISID` or `SAPISID`) and `x-goog-authuser`.

**Never commit `headers.json`.** Mount it as a secret (e.g. `secrets/ytmusic/headers.json`).

## When the Premium session dies

Sessions usually last a long time (often many months) but die when you:

- Sign out of that Google account in the browser that minted the cookies
- Revoke the session from [Google Account → Security → Your devices](https://myaccount.google.com/device-activity)
- Clear site cookies for `youtube.com` / `google.com`
- Hit unusual account / bot checks that invalidate the cookie jar

### Symptoms

- MCP tools return errors mentioning `session expired`, `HTTP 401`, or `HTTP 403`
- `--self-test` fails on `library_smoke` / `liked_smoke` while `search_smoke` still works
- Library endpoints return empty or auth errors that previously worked

Search can keep working without auth; library / liked songs will not.

### Refresh steps

1. In a normal browser, open music.youtube.com and confirm you are still signed in.
2. Re-copy request headers from a fresh authenticated `/browse` call (same steps as setup).
3. Overwrite the headers file the MCP / AI agent host reads:

```bash
./bin/youtube-go-mcp auth --out /path/to/headers.json
./bin/youtube-go-mcp auth --validate /path/to/headers.json
```

4. Restart the MCP process (or agent container) so it reloads `YTMUSIC_HEADERS_PATH`.
5. Run `--self-test` and confirm `liked_smoke=ok` / `library_smoke=ok`.

If refresh fails immediately, try a different `/browse` request after clicking Library, and ensure the copied block includes the full `cookie:` line.

## Ops tips

| Tip | Why |
|---|---|
| Prefer a dedicated browser profile for the agent session | Avoids accidental logout while browsing elsewhere |
| Don’t rotate the headers file mid-request | Restart after replacing secrets |
| Keep `YTMUSIC_HEADERS_PATH` pointed at the mounted secret | Same path your agent host’s compose uses |
| On repeated 429s, raise `YTMUSIC_MIN_REQUEST_INTERVAL_MS` | Gentle spacing reduces rate limits |

## Related env

| Env | Default | Purpose |
|---|---|---|
| `YTMUSIC_HEADERS_PATH` | _(empty)_ | Path to browser headers JSON |
| `YTMUSIC_CLIENT_VERSION` | built-in | Override InnerTube `clientVersion` |
| `YTMUSIC_MIN_REQUEST_INTERVAL_MS` | `250` | Minimum spacing between InnerTube calls |
| `YTMUSIC_MAX_RETRIES` | `3` | Retries after HTTP 429/503 |
