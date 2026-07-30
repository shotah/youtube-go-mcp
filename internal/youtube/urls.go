package youtube

import "strings"

// WatchURL returns https://www.youtube.com/watch?v=…
func WatchURL(videoID string) string {
	id := strings.TrimSpace(videoID)
	if id == "" {
		return ""
	}
	return "https://www.youtube.com/watch?v=" + id
}

// MusicWatchURL returns https://music.youtube.com/watch?v=… (convenience link).
func MusicWatchURL(videoID string) string {
	id := strings.TrimSpace(videoID)
	if id == "" {
		return ""
	}
	return "https://music.youtube.com/watch?v=" + id
}

// PlaylistURL returns https://www.youtube.com/playlist?list=…
func PlaylistURL(playlistID string) string {
	id := strings.TrimSpace(playlistID)
	if id == "" {
		return ""
	}
	return "https://www.youtube.com/playlist?list=" + id
}
