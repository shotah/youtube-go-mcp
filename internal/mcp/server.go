package mcp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/shotah/youtube-go-mcp/internal/youtube"
	"github.com/shotah/youtube-go-mcp/internal/ytmusic"
)

const ServerName = "youtube-go-mcp"

// ServerVersion is set at build time via ldflags (see Makefile / GoReleaser).
var ServerVersion = "dev"

// Server wraps the YouTube Data API v3 client as an MCP tool surface.
type Server struct {
	YT  *youtube.Client
	Log *log.Logger
}

// New creates an MCP server bound to the given Data API client.
func New(yt *youtube.Client) *Server {
	return &Server{
		YT:  yt,
		Log: log.New(os.Stderr, "youtube-go-mcp: ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (s *Server) ready() bool {
	return s != nil && s.YT != nil && s.YT.Tokens != nil
}

// Tool names: {service}_{verb}_{object…} — see ai-gantry docs/mcp-naming.md.
// Hosts expose {server}__{tool} (server id = youtube). Never prefix youtube_.
const (
	ToolVideosSearch           = "videos_search"
	ToolVideosGet              = "videos_get"
	ToolPlaylistsGet           = "playlists_get"
	ToolLibraryListPlaylists   = "library_list_playlists"
	ToolLibraryListLikedVideos = "library_list_liked_videos"
	ToolCastFormatTarget       = "cast_format_target"
)

// RegisteredToolNames returns the MCP tool names in registration order.
func RegisteredToolNames() []string {
	return []string{
		ToolVideosSearch,
		ToolVideosGet,
		ToolPlaylistsGet,
		ToolLibraryListPlaylists,
		ToolLibraryListLikedVideos,
		ToolCastFormatTarget,
	}
}

// Run starts the MCP server over stdio. Logs go to stderr only.
func (s *Server) Run(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolVideosSearch,
		Description: "Search YouTube videos by query. Returns videoId, title, channel, duration, and cast-friendly URLs. Set musicOnly for Music category (10). Requires OAuth.",
	}, s.searchVideos)

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolVideosGet,
		Description: "Get metadata for a YouTube videoId (title, channel, duration, cast URLs). Requires OAuth.",
	}, s.getVideo)

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolPlaylistsGet,
		Description: "List videos in a YouTube playlist by playlist id (PL…, LL…). Requires OAuth.",
	}, s.getPlaylist)

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolLibraryListPlaylists,
		Description: "List playlists owned by the authenticated YouTube channel. Requires OAuth.",
	}, s.listPlaylists)

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolLibraryListLikedVideos,
		Description: "List videos the authenticated user liked on YouTube (thumbs-up). Not YouTube Music Liked Songs. Set musicOnly to keep music-leaning rows (category 10 / Topic / title heuristics). Requires OAuth.",
	}, s.listLikedVideos)

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolCastFormatTarget,
		Description: "Format a videoId into cast-ready fields (videoId, video_id, url). Playback is separate — pass video_id to your Cast/YouTube bridge. Same handoff for music or any video.",
	}, s.formatCastTarget)

	s.Log.Printf("starting stdio MCP (%s %s), auth=%v", ServerName, ServerVersion, s.ready())
	return server.Run(ctx, &mcp.StdioTransport{})
}

type videoOut struct {
	VideoID      string `json:"videoId"`
	VideoIDSnake string `json:"video_id"`
	Title        string `json:"title"`
	ChannelID    string `json:"channelId,omitempty"`
	ChannelTitle string `json:"channelTitle,omitempty"`
	CategoryID   string `json:"categoryId,omitempty"`
	DurationSec  int    `json:"durationSec,omitempty"`
	URL          string `json:"url"`
	MusicURL     string `json:"musicUrl,omitempty"`
	// MusicLikely is true when category / Topic / title heuristics say music.
	MusicLikely bool `json:"musicLikely,omitempty"`
}

func videoToOut(v youtube.Video) videoOut {
	likely := youtube.LooksLikeMusic(v) || v.MusicURL != ""
	return videoOut{
		VideoID:      v.VideoID,
		VideoIDSnake: v.VideoID,
		Title:        v.Title,
		ChannelID:    v.ChannelID,
		ChannelTitle: v.ChannelTitle,
		CategoryID:   v.CategoryID,
		DurationSec:  v.DurationSec,
		URL:          v.URL,
		MusicURL:     v.MusicURL,
		MusicLikely:  likely,
	}
}

type searchVideosInput struct {
	Query     string `json:"query" jsonschema:"YouTube search query"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Max videos to return (default 10, max 50)"`
	MusicOnly bool   `json:"musicOnly,omitempty" jsonschema:"When true, restrict to Music category (videoCategoryId=10)"`
}

type searchVideosOutput struct {
	Videos []videoOut `json:"videos"`
}

func (s *Server) searchVideos(ctx context.Context, _ *mcp.CallToolRequest, in searchVideosInput) (*mcp.CallToolResult, searchVideosOutput, error) {
	_ = ctx
	if !s.ready() {
		return toolErrFrom(ytmusic.ErrAuthRequired), searchVideosOutput{}, nil
	}
	if strings.TrimSpace(in.Query) == "" {
		return toolError("query is required"), searchVideosOutput{}, nil
	}
	limit := clampLimit(in.Limit, 10, 50)

	items, err := s.YT.SearchVideos(youtube.SearchOptions{
		Query:      in.Query,
		MaxResults: limit,
		MusicOnly:  in.MusicOnly,
	})
	if err != nil {
		return toolErrFrom(fmt.Errorf("%s failed: %w", ToolVideosSearch, err)), searchVideosOutput{}, nil
	}
	out := searchVideosOutput{Videos: make([]videoOut, 0, len(items))}
	for i := range items {
		if items[i].VideoID == "" {
			continue
		}
		out.Videos = append(out.Videos, videoToOut(items[i]))
	}
	return nil, out, nil
}

type getVideoInput struct {
	VideoID string `json:"videoId" jsonschema:"YouTube videoId (11 chars)"`
}

type getVideoOutput struct {
	Video videoOut `json:"video"`
}

func (s *Server) getVideo(ctx context.Context, _ *mcp.CallToolRequest, in getVideoInput) (*mcp.CallToolResult, getVideoOutput, error) {
	_ = ctx
	if !s.ready() {
		return toolErrFrom(ytmusic.ErrAuthRequired), getVideoOutput{}, nil
	}
	if strings.TrimSpace(in.VideoID) == "" {
		return toolError("videoId is required"), getVideoOutput{}, nil
	}
	v, err := s.YT.GetVideo(in.VideoID)
	if err != nil {
		return toolErrFrom(fmt.Errorf("%s failed: %w", ToolVideosGet, err)), getVideoOutput{}, nil
	}
	return nil, getVideoOutput{Video: videoToOut(*v)}, nil
}

type getPlaylistInput struct {
	PlaylistID string `json:"playlistId" jsonschema:"Playlist id (PL…, LL…). VL- prefix optional."`
	Limit      int    `json:"limit,omitempty" jsonschema:"Max videos to return (default 50, max 200)"`
}

type playlistOutput struct {
	ID     string     `json:"id"`
	Videos []videoOut `json:"videos"`
}

func (s *Server) getPlaylist(ctx context.Context, _ *mcp.CallToolRequest, in getPlaylistInput) (*mcp.CallToolResult, playlistOutput, error) {
	_ = ctx
	if !s.ready() {
		return toolErrFrom(ytmusic.ErrAuthRequired), playlistOutput{}, nil
	}
	if strings.TrimSpace(in.PlaylistID) == "" {
		return toolError("playlistId is required"), playlistOutput{}, nil
	}
	limit := clampLimit(in.Limit, 50, 200)
	items, err := s.YT.ListPlaylistItems(in.PlaylistID, youtube.ListOptions{MaxResults: limit})
	if err != nil {
		return toolErrFrom(fmt.Errorf("%s failed: %w", ToolPlaylistsGet, err)), playlistOutput{}, nil
	}
	out := playlistOutput{
		ID:     strings.TrimPrefix(strings.TrimSpace(in.PlaylistID), "VL"),
		Videos: make([]videoOut, 0, len(items)),
	}
	for i := range items {
		if items[i].VideoID == "" {
			continue
		}
		out.Videos = append(out.Videos, videoToOut(items[i]))
	}
	return nil, out, nil
}

type libraryPlaylistsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Max playlists to return (default 25)"`
}

type libraryPlaylistOut struct {
	PlaylistID string `json:"playlistId"`
	Title      string `json:"title"`
	ItemCount  int64  `json:"itemCount,omitempty"`
	URL        string `json:"url,omitempty"`
}

type libraryPlaylistsOutput struct {
	Playlists []libraryPlaylistOut `json:"playlists"`
}

func (s *Server) listPlaylists(ctx context.Context, _ *mcp.CallToolRequest, in libraryPlaylistsInput) (*mcp.CallToolResult, libraryPlaylistsOutput, error) {
	_ = ctx
	if !s.ready() {
		return toolErrFrom(ytmusic.ErrAuthRequired), libraryPlaylistsOutput{}, nil
	}
	limit := clampLimit(in.Limit, 25, 50)
	items, err := s.YT.ListMyPlaylists(youtube.ListOptions{MaxResults: limit})
	if err != nil {
		return toolErrFrom(fmt.Errorf("%s failed: %w", ToolLibraryListPlaylists, err)), libraryPlaylistsOutput{}, nil
	}
	out := libraryPlaylistsOutput{Playlists: make([]libraryPlaylistOut, 0, len(items))}
	for i := range items {
		out.Playlists = append(out.Playlists, libraryPlaylistOut{
			PlaylistID: items[i].ID,
			Title:      items[i].Title,
			ItemCount:  items[i].ItemCount,
			URL:        items[i].URL,
		})
	}
	return nil, out, nil
}

type likedVideosInput struct {
	Limit     int  `json:"limit,omitempty" jsonschema:"Max liked videos to return (default 50, max 200)"`
	MusicOnly bool `json:"musicOnly,omitempty" jsonschema:"When true, keep only music-leaning likes (category 10 / Topic / title heuristics)"`
}

type likedVideosOutput struct {
	Videos []videoOut `json:"videos"`
}

func (s *Server) listLikedVideos(ctx context.Context, _ *mcp.CallToolRequest, in likedVideosInput) (*mcp.CallToolResult, likedVideosOutput, error) {
	_ = ctx
	if !s.ready() {
		return toolErrFrom(ytmusic.ErrAuthRequired), likedVideosOutput{}, nil
	}
	limit := clampLimit(in.Limit, 50, 200)
	items, err := s.YT.ListLikedVideos(youtube.ListOptions{MaxResults: limit, MusicOnly: in.MusicOnly})
	if err != nil {
		return toolErrFrom(fmt.Errorf("%s failed: %w", ToolLibraryListLikedVideos, err)), likedVideosOutput{}, nil
	}
	out := likedVideosOutput{Videos: make([]videoOut, 0, len(items))}
	for i := range items {
		if items[i].VideoID == "" {
			continue
		}
		out.Videos = append(out.Videos, videoToOut(items[i]))
	}
	return nil, out, nil
}

type castTargetInput struct {
	VideoID string `json:"videoId" jsonschema:"YouTube videoId"`
}

type castTargetOutput struct {
	VideoID      string `json:"videoId"`
	VideoIDSnake string `json:"video_id"`
	URL          string `json:"url"`
	CastHint     string `json:"castHint"`
}

func (s *Server) formatCastTarget(ctx context.Context, _ *mcp.CallToolRequest, in castTargetInput) (*mcp.CallToolResult, castTargetOutput, error) {
	_ = ctx
	id := strings.TrimSpace(in.VideoID)
	if id == "" {
		return toolError("videoId is required"), castTargetOutput{}, nil
	}
	return nil, castTargetOutput{
		VideoID:      id,
		VideoIDSnake: id,
		URL:          youtube.WatchURL(id),
		CastHint: "Pass video_id to your Cast/YouTube playback bridge " +
			"(e.g. mcp-beam beam_youtube_video or cast__youtube_beam_video). " +
			"Same handoff for a song or any video. Do not pass url as a generic media URL.",
	}, nil
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
		if !strings.Contains(msg, ytmusic.AuthRefreshHint) {
			msg += " | " + ytmusic.AuthRefreshHint
		}
	case errors.Is(err, youtube.ErrAuthRequired), errors.Is(err, ytmusic.ErrAuthRequired):
		msg += "; set YOUTUBE_OAUTH_PATH + client id/secret (docs/auth.md)"
	case errors.Is(err, youtube.ErrAPI):
		msg += " | check OAuth scopes, quota, and Data API enablement"
	}
	return toolError(msg)
}

// SelfTest runs Data API smokes (search + channel/likes) when OAuth is configured.
func SelfTest(ym *ytmusic.Client) error {
	if ym == nil {
		ym = ytmusic.NewClient()
	}
	oauthPath := ytmusic.EnvFirst(ytmusic.EnvOAuthPath, "YTMUSIC_OAUTH_PATH")
	fmt.Fprintf(os.Stderr, "version=%s oauth_ready=%v oauth_path=%q\n",
		ServerVersion, ym.OAuth != nil && ym.OAuth.Ready(), oauthPath)

	if ym.OAuth == nil || !ym.OAuth.Ready() {
		return errors.New("self-test requires OAuth (set YOUTUBE_OAUTH_PATH + client id/secret)")
	}

	yt := youtube.New(ym.OAuth)
	if ym.HTTPClient != nil {
		yt.HTTPClient = ym.HTTPClient
	}

	hits, err := yt.SearchVideos(youtube.SearchOptions{Query: "test", MaxResults: 5})
	if err != nil {
		return fmt.Errorf("search smoke failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "search_smoke=ok videos=%d\n", len(hits))
	if len(hits) == 0 {
		return errors.New("search smoke returned zero videos")
	}

	probe, err := ym.ProbeDataAPI()
	if err != nil {
		return fmt.Errorf("data API smoke failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "data_api_smoke=ok channel=%q liked_videos=%d music_category=%d\n",
		probe.ChannelTitle, probe.LikedVideos, probe.MusicCategoryN)
	if probe.LikedSample != "" {
		fmt.Fprintf(os.Stderr, "sample_liked=%q\n", probe.LikedSample)
	}
	if probe.Hint != "" {
		fmt.Fprintf(os.Stderr, "hint=%s\n", probe.Hint)
	}
	if probe.ChannelID == "" {
		return errors.New("data API smoke: channels.list returned no channel")
	}
	return nil
}
