package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	mcpserver "github.com/shotah/youtube-go-mcp/internal/mcp"
	"github.com/shotah/youtube-go-mcp/internal/youtube"
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
	oauthPath := fs.String("oauth", "", "path to oauth.json (overrides YOUTUBE_OAUTH_PATH)")
	oauthClientID := fs.String("oauth-client-id", "", "OAuth client id (or YOUTUBE_OAUTH_CLIENT_ID)")
	oauthClientSecret := fs.String("oauth-client-secret", "", "OAuth client secret (or YOUTUBE_OAUTH_CLIENT_SECRET)")
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

	authClient := ytmusic.NewClient()
	if err := loadClientAuth(authClient, authLoadOpts{
		oauthPath:     firstFlagOrEnv(*oauthPath, ytmusic.EnvOAuthPath, "YTMUSIC_OAUTH_PATH"),
		oauthClientID: firstFlagOrEnv(*oauthClientID, ytmusic.EnvOAuthClientID, "YTMUSIC_OAUTH_CLIENT_ID"),
		oauthSecret:   firstFlagOrEnv(*oauthClientSecret, ytmusic.EnvOAuthClientSecret, "YTMUSIC_OAUTH_CLIENT_SECRET"),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if *selfTest {
			return 1
		}
	}

	if *selfTest {
		if err := mcpserver.SelfTest(authClient); err != nil {
			fmt.Fprintf(os.Stderr, "self-test failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(os.Stderr, "self-test ok")
		return 0
	}

	var ytClient *youtube.Client
	if authClient.OAuth != nil && authClient.OAuth.Ready() {
		ytClient = youtube.New(authClient.OAuth)
		if authClient.HTTPClient != nil {
			ytClient.HTTPClient = authClient.HTTPClient
		}
	}
	srv := mcpserver.New(ytClient)
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
		case "help", "-h", "--help":
			printAuthUsage(os.Stdout)
			return 0
		case "browser":
			fmt.Fprintln(os.Stderr, "browser cookie auth was removed — use: youtube-go-mcp auth oauth")
			return 2
		}
	}
	printAuthUsage(os.Stdout)
	return 0
}

func runAuthOAuth(args []string) int {
	fs := flag.NewFlagSet("auth oauth", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	outPath := fs.String("out", "oauth.json", "output oauth.json path")
	clientID := fs.String("client-id", "", "Google OAuth client id (or YOUTUBE_OAUTH_CLIENT_ID)")
	clientSecret := fs.String("client-secret", "", "Google OAuth client secret (or YOUTUBE_OAUTH_CLIENT_SECRET)")
	validate := fs.String("validate", "", "validate an existing oauth.json and exit")
	whoami := fs.Bool("whoami", false, "print Google tokeninfo + YouTube channel for configured OAuth and exit")
	probeData := fs.Bool("probe-data-api", false, "Data API v3 smoke: channels.list mine + videos.list myRating=like")
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

	if *whoami || *probeData {
		return runOAuthDiagnostics(oauthDiagOpts{
			outPath:      *outPath,
			clientID:     *clientID,
			clientSecret: *clientSecret,
			whoami:       *whoami,
			probeData:    *probeData,
		})
	}

	creds := ytmusic.OAuthCredentials{
		ClientID:     firstFlagOrEnv(*clientID, ytmusic.EnvOAuthClientID, "YTMUSIC_OAUTH_CLIENT_ID"),
		ClientSecret: firstFlagOrEnv(*clientSecret, ytmusic.EnvOAuthClientSecret, "YTMUSIC_OAUTH_CLIENT_SECRET"),
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
	fmt.Fprintln(os.Stderr, "  YOUTUBE_OAUTH_PATH="+*outPath)
	fmt.Fprintln(os.Stderr, "  YOUTUBE_OAUTH_CLIENT_ID=<same client id>")
	fmt.Fprintln(os.Stderr, "  YOUTUBE_OAUTH_CLIENT_SECRET=<same client secret>")
	fmt.Fprintln(os.Stderr, "(Legacy YTMUSIC_OAUTH_* names still work.)")
	return 0
}

func firstFlagOrEnv(flagVal string, envNames ...string) string {
	if strings.TrimSpace(flagVal) != "" {
		return strings.TrimSpace(flagVal)
	}
	return ytmusic.EnvFirst(envNames...)
}

type oauthDiagOpts struct {
	outPath      string
	clientID     string
	clientSecret string
	whoami       bool
	probeData    bool
}

func runOAuthDiagnostics(opts oauthDiagOpts) int {
	client := ytmusic.NewClient()
	oauthPath := ytmusic.EnvFirst(ytmusic.EnvOAuthPath, "YTMUSIC_OAUTH_PATH")
	if oauthPath == "" {
		oauthPath = opts.outPath
	}
	if err := client.SetOAuthPath(
		oauthPath,
		firstFlagOrEnv(opts.clientID, ytmusic.EnvOAuthClientID, "YTMUSIC_OAUTH_CLIENT_ID"),
		firstFlagOrEnv(opts.clientSecret, ytmusic.EnvOAuthClientSecret, "YTMUSIC_OAUTH_CLIENT_SECRET"),
	); err != nil {
		fmt.Fprintf(os.Stderr, "oauth load failed: %v\n", err)
		return 1
	}
	if opts.whoami {
		info, err := client.WhoAmI()
		if err != nil {
			fmt.Fprintf(os.Stderr, "whoami failed: %v\n", err)
			return 1
		}
		b, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(b))
		if info.Email == "" {
			fmt.Fprintln(os.Stderr, "warning: tokeninfo has no email — youtube scope often omits it; trust channelId/title")
		}
		return 0
	}
	probeResult, err := client.ProbeDataAPI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "probe-data-api failed: %v\n", err)
		if probeResult != nil {
			b, _ := json.MarshalIndent(probeResult, "", "  ")
			fmt.Println(string(b))
		}
		return 1
	}
	b, _ := json.MarshalIndent(probeResult, "", "  ")
	fmt.Println(string(b))
	return 0
}

type authLoadOpts struct {
	oauthPath     string
	oauthClientID string
	oauthSecret   string
}

func loadClientAuth(client *ytmusic.Client, opts authLoadOpts) error {
	if opts.oauthPath == "" {
		return nil
	}
	if err := client.SetOAuthPath(opts.oauthPath, opts.oauthClientID, opts.oauthSecret); err != nil {
		return fmt.Errorf("oauth load failed: %w", err)
	}
	return nil
}

func printOAuthInstructions(w io.Writer) {
	fmt.Fprint(w, `OAuth setup (YouTube Data API v3 — long-lived via refresh_token)

1. Google Cloud Console → create/select a project.
2. Enable "YouTube Data API v3".
3. APIs & Services → Credentials → Create credentials → OAuth client ID
   → Application type: "TVs and Limited Input devices".
4. Copy client id + client secret (or export YOUTUBE_OAUTH_CLIENT_ID / _SECRET).
5. This CLI will print a URL + code; approve in any browser, then wait.

Scope minted: https://www.googleapis.com/auth/youtube
(youtube.readonly also works for reads if you remint with that scope.)

Never commit oauth.json / client secrets.
See docs/auth.md.

`)
}

func printAuthUsage(w io.Writer) {
	fmt.Fprint(w, `youtube-go-mcp auth — credentials helpers

  youtube-go-mcp auth oauth [--out oauth.json] [--client-id ID] [--client-secret SECRET]
  youtube-go-mcp auth oauth --validate oauth.json
  youtube-go-mcp auth oauth --whoami              # tokeninfo + YouTube channel
  youtube-go-mcp auth oauth --probe-data-api      # Data API: channels.mine + liked videos

`)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `youtube-go-mcp — YouTube Data API MCP server (stdio)

Usage:
  youtube-go-mcp [--oauth path] [--oauth-client-id ID] [--oauth-client-secret SECRET]
  youtube-go-mcp --self-test               Smoke-test Data API (requires OAuth)
  youtube-go-mcp --version
  youtube-go-mcp auth oauth                Durable OAuth device flow

Env (preferred):
  YOUTUBE_OAUTH_PATH                Path to oauth.json
  YOUTUBE_OAUTH_CLIENT_ID           Google OAuth client id
  YOUTUBE_OAUTH_CLIENT_SECRET       Google OAuth client secret

Legacy aliases (still accepted): YTMUSIC_OAUTH_*

Tools: videos_search, videos_get, playlists_get, library_list_playlists,
       library_list_liked_videos, cast_format_target

`)
}
