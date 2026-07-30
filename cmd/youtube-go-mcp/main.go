package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	mcpserver "github.com/shotah/youtube-go-mcp/internal/mcp"
	"github.com/shotah/youtube-go-mcp/internal/ytmusic"
)

// version is set at build time via ldflags (see Makefile / GoReleaser).
var version = "dev"

func main() {
	if version != "" && version != "dev" {
		mcpserver.ServerVersion = version
	}
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "auth":
			return runAuth(args[1:])
		case "help", "-h", "--help":
			printUsage(os.Stdout)
			return 0
		}
	}

	fs := flag.NewFlagSet("youtube-go-mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	selfTest := fs.Bool("self-test", false, "run smoke checks and exit")
	headersPath := fs.String("headers", "", "path to browser headers JSON (overrides YTMUSIC_HEADERS_PATH)")
	oauthPath := fs.String("oauth", "", "path to oauth.json (overrides YTMUSIC_OAUTH_PATH)")
	oauthClientID := fs.String("oauth-client-id", "", "OAuth client id (or YTMUSIC_OAUTH_CLIENT_ID)")
	oauthClientSecret := fs.String("oauth-client-secret", "", "OAuth client secret (or YTMUSIC_OAUTH_CLIENT_SECRET)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		if version != "" && version != "dev" {
			fmt.Println(version)
		} else {
			fmt.Println(mcpserver.ServerVersion)
		}
		return 0
	}

	client := ytmusic.NewClient()
	if err := loadClientAuth(client, authLoadOpts{
		oauthPath:     firstFlagOrEnv(*oauthPath, "YTMUSIC_OAUTH_PATH"),
		oauthClientID: firstFlagOrEnv(*oauthClientID, "YTMUSIC_OAUTH_CLIENT_ID"),
		oauthSecret:   firstFlagOrEnv(*oauthClientSecret, "YTMUSIC_OAUTH_CLIENT_SECRET"),
		headersPath:   firstFlagOrEnv(*headersPath, "YTMUSIC_HEADERS_PATH"),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if *selfTest {
			return 1
		}
	} else if client.Authenticated() {
		ytmusic.Default = client
	}

	if *selfTest {
		if err := mcpserver.SelfTest(client); err != nil {
			fmt.Fprintf(os.Stderr, "self-test failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(os.Stderr, "self-test ok")
		return 0
	}

	srv := mcpserver.New(client)
	if err := srv.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
		return 1
	}
	return 0
}

func runAuth(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "oauth":
			return runAuthOAuth(args[1:])
		case "browser":
			return runAuthBrowser(args[1:])
		case "help", "-h", "--help":
			printAuthUsage(os.Stdout)
			return 0
		}
	}
	// Default: browser flow (back-compat). Prefer: auth oauth
	return runAuthBrowser(args)
}

func runAuthOAuth(args []string) int {
	fs := flag.NewFlagSet("auth oauth", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	outPath := fs.String("out", "oauth.json", "output oauth.json path")
	clientID := fs.String("client-id", "", "Google OAuth client id (or YTMUSIC_OAUTH_CLIENT_ID)")
	clientSecret := fs.String("client-secret", "", "Google OAuth client secret (or YTMUSIC_OAUTH_CLIENT_SECRET)")
	validate := fs.String("validate", "", "validate an existing oauth.json and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *validate != "" {
		tok, err := ytmusic.LoadOAuthTokenFromFile(*validate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "valid oauth.json (refresh_token present, expires_at=%d)\n", tok.ExpiresAt)
		return 0
	}

	creds := ytmusic.OAuthCredentials{
		ClientID:     firstFlagOrEnv(*clientID, "YTMUSIC_OAUTH_CLIENT_ID"),
		ClientSecret: firstFlagOrEnv(*clientSecret, "YTMUSIC_OAUTH_CLIENT_SECRET"),
	}
	printOAuthInstructions(os.Stderr)

	tok, err := ytmusic.RunDeviceAuthFlow(context.Background(), creds, func(code *ytmusic.DeviceCode) error {
		fmt.Fprintf(os.Stderr, "\nOpen: %s\nEnter code: %s\n\nWaiting for Google authorization (Ctrl-C to abort)…\n",
			code.VerificationLink(), code.UserCode)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "oauth failed: %v\n", err)
		return 1
	}
	if err := tok.Save(*outPath); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *outPath)
	fmt.Fprintln(os.Stderr, "Set all three (never commit the json):")
	fmt.Fprintln(os.Stderr, "  YTMUSIC_OAUTH_PATH="+*outPath)
	fmt.Fprintln(os.Stderr, "  YTMUSIC_OAUTH_CLIENT_ID=<same client id>")
	fmt.Fprintln(os.Stderr, "  YTMUSIC_OAUTH_CLIENT_SECRET=<same client secret>")
	return 0
}

func runAuthBrowser(args []string) int {
	fs := flag.NewFlagSet("auth browser", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	outPath := fs.String("out", "headers.json", "output headers JSON path")
	validate := fs.String("validate", "", "validate an existing headers JSON file and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *validate != "" {
		auth, err := ytmusic.LoadAuthFromFile(*validate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "valid headers for authuser=%s (SAPISID present)\n", auth.AuthUser)
		return 0
	}

	printAuthInstructions(os.Stderr)

	in := bufio.NewReader(os.Stdin)
	cookie, err := promptLine(in, os.Stderr, "cookie")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read cookie: %v\n", err)
		return 1
	}
	authUser, err := promptLine(in, os.Stderr, "x-goog-authuser")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read x-goog-authuser: %v\n", err)
		return 1
	}

	headers, err := headersFromPrompts(cookie, authUser)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	data, err := json.MarshalIndent(headers, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		return 1
	}
	if _, err := ytmusic.ParseAuthHeaders(data); err != nil {
		fmt.Fprintf(os.Stderr, "headers incomplete: %v\n", err)
		return 1
	}
	if err := os.WriteFile(*outPath, append(data, '\n'), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "wrote %s — set YTMUSIC_HEADERS_PATH to this file (never commit it)\n", *outPath)
	fmt.Fprintln(os.Stderr, "Tip: prefer durable OAuth via: youtube-go-mcp auth oauth")
	return 0
}

func firstFlagOrEnv(flagVal, envName string) string {
	if strings.TrimSpace(flagVal) != "" {
		return strings.TrimSpace(flagVal)
	}
	return strings.TrimSpace(os.Getenv(envName))
}

type authLoadOpts struct {
	oauthPath     string
	oauthClientID string
	oauthSecret   string
	headersPath   string
}

func loadClientAuth(client *ytmusic.Client, opts authLoadOpts) error {
	if opts.oauthPath != "" {
		if err := client.SetOAuthPath(opts.oauthPath, opts.oauthClientID, opts.oauthSecret); err != nil {
			return fmt.Errorf("oauth load failed: %w", err)
		}
		return nil
	}
	if opts.headersPath == "" {
		return nil
	}
	if err := client.SetAuthPath(opts.headersPath); err != nil {
		return fmt.Errorf("auth load failed: %w", err)
	}
	return nil
}

func promptLine(in *bufio.Reader, errOut io.Writer, name string) (string, error) {
	fmt.Fprintf(errOut, "%s: ", name)
	line, err := in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// headersFromPrompts builds a headers JSON map from the two values DevTools shows
// under Request Headers. Accepts bare values or "Name: value" / "name: value" lines.
func headersFromPrompts(cookie, authUser string) (map[string]string, error) {
	cookie = stripHeaderPrefix(cookie, "cookie")
	authUser = stripHeaderPrefix(authUser, "x-goog-authuser")
	if cookie == "" || authUser == "" {
		return nil, errors.New("need both cookie and x-goog-authuser (copy each value from Request Headers on a /browse call)")
	}
	return map[string]string{
		"cookie":          cookie,
		"x-goog-authuser": authUser,
		"content-type":    "application/json",
		"x-origin":        "https://music.youtube.com",
	}, nil
}

func stripHeaderPrefix(raw, name string) string {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	prefix := strings.ToLower(name) + ":"
	if strings.HasPrefix(lower, prefix) {
		return strings.TrimSpace(raw[len(prefix):])
	}
	return raw
}

func printOAuthInstructions(w io.Writer) {
	fmt.Fprint(w, `OAuth setup (preferred — survives normal YouTube browser logins)

1. Google Cloud Console → create/select a project.
2. Enable "YouTube Data API v3".
3. APIs & Services → Credentials → Create credentials → OAuth client ID
   → Application type: "TVs and Limited Input devices".
4. Copy client id + client secret (or export YTMUSIC_OAUTH_CLIENT_ID / _SECRET).
5. This CLI will print a URL + code; approve in any browser, then wait.

Never commit oauth.json / client secrets.
See docs/auth.md.

`)
}

func printAuthInstructions(w io.Writer) {
	fmt.Fprint(w, `Browser auth setup (legacy — dies when that browser session logs out)

Prefer: youtube-go-mcp auth oauth

1. Open https://music.youtube.com and sign in.
2. DevTools (F12) → Network → filter "browse".
3. Click Library (or scroll) so a POST to /youtubei/v1/browse appears.
4. Click that request → Headers → Request Headers.
5. Copy the value of cookie (long string; must include __Secure-3PAPISID).
6. Copy the value of x-goog-authuser (usually 0).
7. Paste each when prompted below (Enter after each).

Use a dedicated browser profile only for minting cookies — never your daily login.
Never commit headers.json / cookies. See docs/auth.md.

`)
}

func printAuthUsage(w io.Writer) {
	fmt.Fprint(w, `youtube-go-mcp auth — credentials helpers

  youtube-go-mcp auth oauth [--out oauth.json] [--client-id ID] [--client-secret SECRET]
  youtube-go-mcp auth oauth --validate oauth.json
  youtube-go-mcp auth browser [--out headers.json]
  youtube-go-mcp auth browser --validate headers.json

`)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `youtube-go-mcp — YouTube Music MCP server (stdio)

Usage:
  youtube-go-mcp [--oauth path] [--oauth-client-id ID] [--oauth-client-secret SECRET]
  youtube-go-mcp [--headers path]          Browser-cookie fallback
  youtube-go-mcp --self-test               Smoke-test search (+ library if authed)
  youtube-go-mcp --version
  youtube-go-mcp auth oauth                Durable OAuth device flow (preferred)
  youtube-go-mcp auth browser              Legacy browser cookie export

Env:
  YTMUSIC_OAUTH_PATH                Path to oauth.json (preferred)
  YTMUSIC_OAUTH_CLIENT_ID           Google OAuth client id
  YTMUSIC_OAUTH_CLIENT_SECRET       Google OAuth client secret
  YTMUSIC_HEADERS_PATH              Path to browser headers JSON (legacy)
  YTMUSIC_CLIENT_VERSION            Override InnerTube clientVersion
  YTMUSIC_MIN_REQUEST_INTERVAL_MS   Min spacing between calls (default 250)
  YTMUSIC_MAX_RETRIES               Retries on HTTP 429/503 (default 3)

`)
}
