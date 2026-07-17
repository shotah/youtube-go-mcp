package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/shotah/youtube-go-mcp/internal/ytmusic"
)

const ServerName = "youtube-go-mcp"

// ServerVersion is set at build time via ldflags (see Makefile / GoReleaser).
var ServerVersion = "dev"

// Server wraps the YouTube Music client as an MCP tool surface.
type Server struct {
	Client *ytmusic.Client
	Log    *log.Logger
}

// New creates an MCP server bound to the given ytmusic client.
func New(client *ytmusic.Client) *Server {
	if client == nil {
		client = ytmusic.NewClient()
	}
	return &Server{
		Client: client,
		Log:    log.New(os.Stderr, "youtube-go-mcp: ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Run starts the MCP server over stdio. Logs go to stderr only.
func (s *Server) Run(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_tracks",
		Description: "Search YouTube Music for tracks. Returns videoId, title, artists, duration, and cast-friendly URLs.",
	}, s.searchTracks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_library_playlists",
		Description: "List playlists from the authenticated YouTube Music library. Requires browser session headers.",
	}, s.getLibraryPlaylists)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_playlist",
		Description: "Get tracks from a YouTube Music playlist by playlist id (PL…, RD…, or LM for Liked Songs).",
	}, s.getPlaylist)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_liked_songs",
		Description: "Get tracks from the authenticated user's Liked Songs. Useful for taste-aware suggestions. Requires browser session headers.",
	}, s.getLikedSongs)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_history",
		Description: "Get recently played YouTube Music tracks (listening history) for continuity and suggestions. Requires browser session headers.",
	}, s.getHistory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_watch_playlist",
		Description: "Get a radio / continuum playlist seeded from a videoId.",
	}, s.getWatchPlaylist)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_track",
		Description: "Get metadata for a videoId (title, artists, duration, cast URLs). Set includeLyrics to also fetch lyrics when available — useful for understanding the song.",
	}, s.getTrack)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_lyrics",
		Description: "Get plain-text lyrics for a videoId when YouTube Music provides them. Returns available=false when the track has no lyrics.",
	}, s.getLyrics)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "format_cast_target",
		Description: "Format a videoId into the payload Cast / Nest (or similar) integrations expect.",
	}, s.formatCastTarget)

	s.Log.Printf("starting stdio MCP (%s %s), auth=%v", ServerName, ServerVersion, s.Client.Authenticated())
	return server.Run(ctx, &mcp.StdioTransport{})
}

type searchTracksInput struct {
	Query string `json:"query" jsonschema:"YouTube Music search query"`
	Limit int    `json:"limit,omitempty" jsonschema:"Max tracks to return (default 10, max 50)"`
}

type trackOut struct {
	VideoID    string   `json:"videoId"`
	PlaylistID string   `json:"playlistId,omitempty"`
	Title      string   `json:"title"`
	Artists    []string `json:"artists"`
	Duration   int      `json:"duration"`
	IsExplicit bool     `json:"isExplicit"`
	URL        string   `json:"url"`
	MusicURL   string   `json:"musicUrl"`
}

type searchTracksOutput struct {
	Tracks []trackOut `json:"tracks"`
}

func (s *Server) searchTracks(ctx context.Context, _ *mcp.CallToolRequest, in searchTracksInput) (*mcp.CallToolResult, searchTracksOutput, error) {
	_ = ctx
	if in.Query == "" {
		return toolError("query is required"), searchTracksOutput{}, nil
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	result, err := s.Client.TrackSearch(in.Query).Next()
	if err != nil {
		return toolErrFrom(fmt.Errorf("search failed: %w", err)), searchTracksOutput{}, nil
	}

	out := searchTracksOutput{Tracks: make([]trackOut, 0, limit)}
	for i, t := range result.Tracks {
		if i >= limit {
			break
		}
		out.Tracks = append(out.Tracks, trackToOut(t))
	}
	return nil, out, nil
}

type libraryPlaylistsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Max playlists to return (default 25)"`
}

type libraryPlaylistOut struct {
	PlaylistID string `json:"playlistId"`
	Title      string `json:"title"`
	Count      string `json:"count,omitempty"`
}

type libraryPlaylistsOutput struct {
	Playlists []libraryPlaylistOut `json:"playlists"`
}

func (s *Server) getLibraryPlaylists(ctx context.Context, _ *mcp.CallToolRequest, in libraryPlaylistsInput) (*mcp.CallToolResult, libraryPlaylistsOutput, error) {
	_ = ctx
	if !s.Client.Authenticated() {
		return toolErrFrom(ytmusic.ErrAuthRequired), libraryPlaylistsOutput{}, nil
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 25
	}

	playlists, err := s.Client.GetLibraryPlaylists(limit)
	if err != nil {
		return toolErrFrom(fmt.Errorf("get_library_playlists failed: %w", err)), libraryPlaylistsOutput{}, nil
	}

	out := libraryPlaylistsOutput{Playlists: make([]libraryPlaylistOut, 0, len(playlists))}
	for _, p := range playlists {
		out.Playlists = append(out.Playlists, libraryPlaylistOut{
			PlaylistID: p.PlaylistID,
			Title:      p.Title,
			Count:      p.Count,
		})
	}
	return nil, out, nil
}

type getPlaylistInput struct {
	PlaylistID string `json:"playlistId" jsonschema:"Playlist id (PL…, RD…, LM for Liked Songs). VL- prefix optional."`
	Limit      int    `json:"limit,omitempty" jsonschema:"Max tracks to return (default 50, max 200)"`
}

type playlistOutput struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Author     string     `json:"author,omitempty"`
	TrackCount int        `json:"trackCount,omitempty"`
	Tracks     []trackOut `json:"tracks"`
}

func (s *Server) getPlaylist(ctx context.Context, _ *mcp.CallToolRequest, in getPlaylistInput) (*mcp.CallToolResult, playlistOutput, error) {
	_ = ctx
	if in.PlaylistID == "" {
		return toolError("playlistId is required"), playlistOutput{}, nil
	}
	limit := clampLimit(in.Limit, 50, 200)

	detail, err := s.Client.GetPlaylist(in.PlaylistID, limit)
	if err != nil {
		return toolErrFrom(fmt.Errorf("get_playlist failed: %w", err)), playlistOutput{}, nil
	}
	return nil, playlistToOut(detail), nil
}

type likedSongsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Max liked tracks to return (default 50, max 200)"`
}

func (s *Server) getLikedSongs(ctx context.Context, _ *mcp.CallToolRequest, in likedSongsInput) (*mcp.CallToolResult, playlistOutput, error) {
	_ = ctx
	if !s.Client.Authenticated() {
		return toolErrFrom(ytmusic.ErrAuthRequired), playlistOutput{}, nil
	}
	limit := clampLimit(in.Limit, 50, 200)

	detail, err := s.Client.GetLikedSongs(limit)
	if err != nil {
		return toolErrFrom(fmt.Errorf("get_liked_songs failed: %w", err)), playlistOutput{}, nil
	}
	return nil, playlistToOut(detail), nil
}

type historyInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Max history items to return (default 50, max 200)"`
}

type historyTrackOut struct {
	trackOut
	Played string `json:"played,omitempty"`
}

type historyOutput struct {
	Tracks []historyTrackOut `json:"tracks"`
}

func (s *Server) getHistory(ctx context.Context, _ *mcp.CallToolRequest, in historyInput) (*mcp.CallToolResult, historyOutput, error) {
	_ = ctx
	if !s.Client.Authenticated() {
		return toolErrFrom(ytmusic.ErrAuthRequired), historyOutput{}, nil
	}
	limit := clampLimit(in.Limit, 50, 200)

	items, err := s.Client.GetHistory(limit)
	if err != nil {
		return toolErrFrom(fmt.Errorf("get_history failed: %w", err)), historyOutput{}, nil
	}

	out := historyOutput{Tracks: make([]historyTrackOut, 0, len(items))}
	for _, item := range items {
		if item == nil || item.VideoID == "" {
			continue
		}
		out.Tracks = append(out.Tracks, historyTrackOut{
			trackOut: trackToOut(&ytmusic.TrackItem{
				VideoID:    item.VideoID,
				PlaylistID: item.PlaylistID,
				Title:      item.Title,
				Artists:    item.Artists,
				Album:      item.Album,
				Duration:   item.Duration,
				IsExplicit: item.IsExplicit,
				Thumbnails: item.Thumbnails,
			}),
			Played: item.Played,
		})
	}
	return nil, out, nil
}

type watchPlaylistInput struct {
	VideoID string `json:"videoId" jsonschema:"Seed track videoId"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max tracks to return (default 25)"`
}

type watchPlaylistOutput struct {
	Tracks []trackOut `json:"tracks"`
}

func (s *Server) getWatchPlaylist(ctx context.Context, _ *mcp.CallToolRequest, in watchPlaylistInput) (*mcp.CallToolResult, watchPlaylistOutput, error) {
	_ = ctx
	if in.VideoID == "" {
		return toolError("videoId is required"), watchPlaylistOutput{}, nil
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 25
	}

	tracks, err := s.Client.GetWatchPlaylist(in.VideoID)
	if err != nil {
		return toolErrFrom(fmt.Errorf("get_watch_playlist failed: %w", err)), watchPlaylistOutput{}, nil
	}

	out := watchPlaylistOutput{Tracks: make([]trackOut, 0, limit)}
	for i, t := range tracks {
		if i >= limit {
			break
		}
		if t == nil || t.VideoID == "" {
			continue
		}
		out.Tracks = append(out.Tracks, trackToOut(t))
	}
	return nil, out, nil
}

type getTrackInput struct {
	VideoID       string `json:"videoId" jsonschema:"YouTube Music videoId"`
	IncludeLyrics bool   `json:"includeLyrics,omitempty" jsonschema:"When true, also fetch lyrics if available (default false)"`
}

type trackDetailOut struct {
	trackOut
	Album     string `json:"album,omitempty"`
	Lyrics    string `json:"lyrics,omitempty"`
	HasLyrics bool   `json:"hasLyrics"`
}

func (s *Server) getTrack(ctx context.Context, _ *mcp.CallToolRequest, in getTrackInput) (*mcp.CallToolResult, trackDetailOut, error) {
	_ = ctx
	if in.VideoID == "" {
		return toolError("videoId is required"), trackDetailOut{}, nil
	}

	detail, err := s.Client.GetTrack(in.VideoID, in.IncludeLyrics)
	if err != nil {
		return toolErrFrom(fmt.Errorf("get_track failed: %w", err)), trackDetailOut{}, nil
	}
	return nil, trackDetailToOut(detail), nil
}

type getLyricsInput struct {
	VideoID string `json:"videoId" jsonschema:"YouTube Music videoId"`
}

type lyricsOutput struct {
	VideoID   string `json:"videoId"`
	Lyrics    string `json:"lyrics,omitempty"`
	Available bool   `json:"available"`
}

func (s *Server) getLyrics(ctx context.Context, _ *mcp.CallToolRequest, in getLyricsInput) (*mcp.CallToolResult, lyricsOutput, error) {
	_ = ctx
	if in.VideoID == "" {
		return toolError("videoId is required"), lyricsOutput{}, nil
	}

	lyrics, err := s.Client.GetLyrics(in.VideoID)
	if errors.Is(err, ytmusic.ErrNoLyrics) {
		return nil, lyricsOutput{VideoID: in.VideoID, Available: false}, nil
	}
	if err != nil {
		return toolErrFrom(fmt.Errorf("get_lyrics failed: %w", err)), lyricsOutput{}, nil
	}
	return nil, lyricsOutput{
		VideoID:   in.VideoID,
		Lyrics:    lyrics,
		Available: lyrics != "",
	}, nil
}

type castTargetInput struct {
	VideoID string `json:"videoId" jsonschema:"YouTube / YouTube Music videoId"`
}

type castTargetOutput struct {
	VideoID  string `json:"videoId"`
	URL      string `json:"url"`
	MusicURL string `json:"musicUrl"`
	// CastHint describes how a Cast MCP should target YouTube receivers.
	CastHint string `json:"castHint"`
}

func (s *Server) formatCastTarget(ctx context.Context, _ *mcp.CallToolRequest, in castTargetInput) (*mcp.CallToolResult, castTargetOutput, error) {
	_ = ctx
	if in.VideoID == "" {
		return toolError("videoId is required"), castTargetOutput{}, nil
	}
	return nil, castTargetOutput{
		VideoID:  in.VideoID,
		URL:      "https://www.youtube.com/watch?v=" + in.VideoID,
		MusicURL: "https://music.youtube.com/watch?v=" + in.VideoID,
		CastHint: "Cast by videoId to a YouTube Cast receiver (not a raw media URL).",
	}, nil
}

func trackDetailToOut(d *ytmusic.TrackDetail) trackDetailOut {
	artists := make([]string, 0, len(d.Artists))
	for _, a := range d.Artists {
		if a.Name != "" {
			artists = append(artists, a.Name)
		}
	}
	return trackDetailOut{
		trackOut: trackOut{
			VideoID:    d.VideoID,
			PlaylistID: d.PlaylistID,
			Title:      d.Title,
			Artists:    artists,
			Duration:   d.Duration,
			IsExplicit: d.IsExplicit,
			URL:        "https://www.youtube.com/watch?v=" + d.VideoID,
			MusicURL:   "https://music.youtube.com/watch?v=" + d.VideoID,
		},
		Album:     d.Album.Name,
		Lyrics:    d.Lyrics,
		HasLyrics: d.HasLyrics,
	}
}

func trackToOut(t *ytmusic.TrackItem) trackOut {
	artists := make([]string, 0, len(t.Artists))
	for _, a := range t.Artists {
		if a.Name != "" {
			artists = append(artists, a.Name)
		}
	}
	return trackOut{
		VideoID:    t.VideoID,
		PlaylistID: t.PlaylistID,
		Title:      t.Title,
		Artists:    artists,
		Duration:   t.Duration,
		IsExplicit: t.IsExplicit,
		URL:        "https://www.youtube.com/watch?v=" + t.VideoID,
		MusicURL:   "https://music.youtube.com/watch?v=" + t.VideoID,
	}
}

func playlistToOut(detail *ytmusic.PlaylistDetail) playlistOutput {
	out := playlistOutput{
		ID:         detail.ID,
		Title:      detail.Title,
		Author:     detail.Author,
		TrackCount: detail.TrackCount,
		Tracks:     make([]trackOut, 0, len(detail.Tracks)),
	}
	for _, t := range detail.Tracks {
		if t == nil || t.VideoID == "" {
			continue
		}
		out.Tracks = append(out.Tracks, trackToOut(t))
	}
	return out
}

func clampLimit(limit, defaultLimit, maxLimit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

func toolErrFrom(err error) *mcp.CallToolResult {
	msg := err.Error()
	switch {
	case errors.Is(err, ytmusic.ErrSessionExpired), errors.Is(err, ytmusic.ErrInvalidAuth):
		msg += " | " + ytmusic.AuthRefreshHint
	case errors.Is(err, ytmusic.ErrRateLimited):
		msg += " | slow down or raise YTMUSIC_MIN_REQUEST_INTERVAL_MS / wait and retry"
	case errors.Is(err, ytmusic.ErrAuthRequired):
		msg += "; export headers from music.youtube.com and set YTMUSIC_HEADERS_PATH (docs/auth.md)"
	}
	return toolError(msg)
}

// SelfTest runs a quick smoke check (auth presence + optional search).
func SelfTest(client *ytmusic.Client) error {
	if client == nil {
		client = ytmusic.NewClient()
	}
	fmt.Fprintf(os.Stderr, "version=%s auth=%v headers_path=%q\n",
		ServerVersion, client.Authenticated(), os.Getenv("YTMUSIC_HEADERS_PATH"))

	result, err := client.TrackSearch("test").Next()
	if err != nil {
		return fmt.Errorf("search smoke failed: %w", err)
	}
	n := len(result.Tracks)
	fmt.Fprintf(os.Stderr, "search_smoke=ok tracks=%d\n", n)
	if n == 0 {
		return errors.New("search smoke returned zero tracks")
	}

	if !client.Authenticated() {
		fmt.Fprintln(os.Stderr, "library_smoke=skipped (no auth)")
		fmt.Fprintln(os.Stderr, "liked_smoke=skipped (no auth)")
		fmt.Fprintln(os.Stderr, "history_smoke=skipped (no auth)")
		return nil
	}
	return selfTestAuthed(client)
}

func selfTestAuthed(client *ytmusic.Client) error {
	playlists, err := client.GetLibraryPlaylists(5)
	if err != nil {
		return fmt.Errorf("library smoke failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "library_smoke=ok playlists=%d\n", len(playlists))
	if len(playlists) > 0 {
		b, _ := json.Marshal(playlists[0])
		fmt.Fprintf(os.Stderr, "sample_playlist=%s\n", b)
	}

	liked, err := client.GetLikedSongs(5)
	if err != nil {
		return fmt.Errorf("liked songs smoke failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "liked_smoke=ok title=%q tracks=%d\n", liked.Title, len(liked.Tracks))
	if len(liked.Tracks) > 0 {
		b, _ := json.Marshal(trackToOut(liked.Tracks[0]))
		fmt.Fprintf(os.Stderr, "sample_liked=%s\n", b)
	}

	history, err := client.GetHistory(5)
	if err != nil {
		return fmt.Errorf("history smoke failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "history_smoke=ok tracks=%d\n", len(history))
	if len(history) > 0 {
		b, _ := json.Marshal(history[0])
		fmt.Fprintf(os.Stderr, "sample_history=%s\n", b)
	}
	return nil
}
