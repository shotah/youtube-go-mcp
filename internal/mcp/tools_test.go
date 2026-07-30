package mcp

import (
	"strings"
	"testing"
)

func TestRegisteredToolNames(t *testing.T) {
	names := RegisteredToolNames()
	want := []string{
		"tracks_search",
		"library_list_playlists",
		"playlists_get",
		"library_list_liked_songs",
		"library_list_history",
		"tracks_list_watch_playlist",
		"tracks_get",
		"tracks_get_lyrics",
		"cast_format_target",
	}
	if len(names) != len(want) {
		t.Fatalf("got %d tools, want %d: %v", len(names), len(want), names)
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("tool[%d]=%q want %q", i, name, want[i])
		}
		if strings.HasPrefix(name, "youtube_") {
			t.Errorf("%q must not include server id prefix", name)
		}
		parts := strings.Split(name, "_")
		if len(parts) < 2 {
			t.Errorf("%q should be {service}_{verb}_{object…}", name)
		}
	}
}
