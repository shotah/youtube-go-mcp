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
	path := *headersPath
	if path == "" {
		path = os.Getenv("YTMUSIC_HEADERS_PATH")
	}
	if path != "" {
		auth, err := ytmusic.LoadAuthFromFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "auth load failed: %v\n", err)
			if *selfTest {
				return 1
			}
		} else {
			client.Auth = auth
			ytmusic.Default = client
		}
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
	fs := flag.NewFlagSet("auth", flag.ContinueOnError)
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
	return 0
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

func printAuthInstructions(w io.Writer) {
	fmt.Fprint(w, `Browser auth setup (YouTube Music Premium session)

1. Open https://music.youtube.com and sign in.
2. DevTools (F12) → Network → filter "browse".
3. Click Library (or scroll) so a POST to /youtubei/v1/browse appears.
4. Click that request → Headers → Request Headers.
5. Copy the value of cookie (long string; must include __Secure-3PAPISID).
6. Copy the value of x-goog-authuser (usually 0).
7. Paste each when prompted below (Enter after each).

Never commit headers.json / cookies.

When library/liked tools break later (session expired / HTTP 401-403):
  re-run this command, overwrite the headers file, restart the MCP.
  See docs/auth.md for the full refresh checklist.

`)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `youtube-go-mcp — YouTube Music MCP server (stdio)

Usage:
  youtube-go-mcp [--headers path]          Run MCP on stdio
  youtube-go-mcp --self-test               Smoke-test search (+ library if authed)
  youtube-go-mcp --version
  youtube-go-mcp auth [--out headers.json] Interactive headers export
  youtube-go-mcp auth --validate FILE      Validate headers JSON

Env:
  YTMUSIC_HEADERS_PATH              Path to browser headers JSON
  YTMUSIC_CLIENT_VERSION            Override InnerTube clientVersion
  YTMUSIC_MIN_REQUEST_INTERVAL_MS   Min spacing between calls (default 250)
  YTMUSIC_MAX_RETRIES               Retries on HTTP 429/503 (default 3)

`)
}
