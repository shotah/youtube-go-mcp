# Browser auth & cookie refresh

YouTube Music library tools (`get_library_playlists`, `get_liked_songs`, private playlists) use your **browser session**, not a Google API key.

## One-time setup

1. Open [music.youtube.com](https://music.youtube.com) and sign in (Premium session recommended).
2. DevTools (**F12**) → **Network** → filter `browse`.
3. Open **Library** (or scroll) so a `POST` to `/youtubei/v1/browse` appears.
4. Click that request → **Headers** → **Request Headers**.
5. Copy these two values (click the value text → Ctrl+C):
   - **`cookie`** — long string; must include `__Secure-3PAPISID` or `SAPISID`
   - **`x-goog-authuser`** — usually `0`
6. Export:

```bash
./bin/youtube-go-mcp auth --out headers.json
# prompts:
#   cookie: <paste, Enter>
#   x-goog-authuser: <paste, Enter>
export YTMUSIC_HEADERS_PATH=$PWD/headers.json
./bin/youtube-go-mcp auth --validate "$YTMUSIC_HEADERS_PATH"
./bin/youtube-go-mcp --self-test
```

You can paste either the bare value or a `Name: value` line — the CLI strips the prefix.

**Never commit `headers.json`.** Mount it as a secret (e.g. `secrets/ytmusic/headers.json`).

> Tip: Chrome often has no “Copy request headers” menu item. Copying the two Request Header values above is enough.

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
2. Re-copy `cookie` and `x-goog-authuser` from a fresh authenticated `/browse` call.
3. Overwrite the headers file:

```bash
./bin/youtube-go-mcp auth --out /path/to/headers.json
./bin/youtube-go-mcp auth --validate /path/to/headers.json
```

4. The MCP **reloads `headers.json` automatically** when the file’s modification time changes (no restart required for a normal overwrite). Restart only if the process was started without `YTMUSIC_HEADERS_PATH` / `--headers`.
5. Run `--self-test` and confirm `liked_smoke=ok` / `library_smoke=ok`.

## Ops tips

| Tip | Why |
|---|---|
| Prefer a dedicated browser profile for the agent session | Avoids accidental logout while browsing elsewhere |
| Overwrite the same mounted headers path in place | The process watches mtime and picks up the new cookie on the next request |
| Keep `YTMUSIC_HEADERS_PATH` pointed at the mounted secret | Same path your agent host’s compose uses |
| On repeated 429s, raise `YTMUSIC_MIN_REQUEST_INTERVAL_MS` | Gentle spacing reduces rate limits |
| Prefer cookies that include `__Secure-3PAPISID` | Used for SAPISIDHASH (falls back to `SAPISID`) |

## Related env

| Env | Default | Purpose |
|---|---|---|
| `YTMUSIC_HEADERS_PATH` | _(empty)_ | Path to browser headers JSON |
| `YTMUSIC_CLIENT_VERSION` | built-in | Override InnerTube `clientVersion` |
| `YTMUSIC_MIN_REQUEST_INTERVAL_MS` | `250` | Minimum spacing between InnerTube calls |
| `YTMUSIC_MAX_RETRIES` | `3` | Retries after HTTP 429/503 |
