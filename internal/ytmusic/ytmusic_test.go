package ytmusic

import (
	"errors"
	"testing"
)

func TestSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("live InnerTube")
	}
	s := Search("ncs")
	r, err := s.Next()
	if err != nil {
		t.Fatal(err)
	}

	if len(r.Tracks) == 0 {
		t.Fatal("len(r.Tracks) == 0")
	}
	if r.Tracks[0].VideoID == "" {
		t.Fatal("r.Tracks[0].VideoID == \"\"")
	}
	if len(r.Tracks[0].Artists) == 0 {
		t.Fatal("len(r.Tracks[0].Artists) == 0")
	}

	if len(r.Videos) == 0 {
		t.Fatal("len(r.Videos) == 0")
	}
	if r.Videos[0].VideoID == "" {
		t.Fatal("r.Videos[0].VideoID == \"\"")
	}

	if len(r.Artists) == 0 {
		t.Fatal("len(r.Artists) == 0")
	}
	if r.Artists[0].BrowseID == "" {
		t.Fatal("r.Artists[0].BrowseID == \"\"")
	}

	if len(r.Playlists) == 0 {
		t.Fatal("len(r.Playlists) == 0")
	}
	if r.Playlists[0].BrowseID == "" {
		t.Fatal("r.Playlists[0].BrowseID == \"\"")
	}

	if len(r.Albums) == 0 {
		t.Fatal("len(r.Albums) == 0")
	}
	if r.Albums[0].BrowseID == "" {
		t.Fatal("r.Albums[0].BrowseID == \"\"")
	}
}

func TestLyrics(t *testing.T) {
	if testing.Short() {
		t.Skip("live InnerTube")
	}
	lyrics, err := GetLyrics("GICwp59Hags")
	if err != nil {
		t.Fatal(err)
	}
	if lyrics == "" {
		t.Fatal("lyrics == \"\"")
	}

	lyrics, err = GetLyrics("9Mf6f8TPH_4")
	if err != nil && !errors.Is(err, ErrNoLyrics) {
		t.Fatal(err)
	}
	if lyrics != "" {
		t.Fatal("lyrics != \"\"")
	}
}

func TestGetTrackLive(t *testing.T) {
	if testing.Short() {
		t.Skip("live InnerTube")
	}
	detail, err := GetTrack("GICwp59Hags", true)
	if err != nil {
		t.Fatal(err)
	}
	if detail.VideoID == "" || detail.Title == "" {
		t.Fatalf("incomplete: %+v", detail)
	}
	if !detail.HasLyrics || detail.Lyrics == "" {
		t.Fatalf("expected lyrics: has=%v len=%d", detail.HasLyrics, len(detail.Lyrics))
	}
	t.Logf("title=%q artists=%v lyrics_len=%d", detail.Title, detail.Artists, len(detail.Lyrics))
}

func TestWatchPlaylist(t *testing.T) {
	if testing.Short() {
		t.Skip("live InnerTube")
	}
	watchPlaylist, err := GetWatchPlaylist("FM7MFYoylVs")
	if err != nil {
		t.Fatal(err)
	}

	if len(watchPlaylist) == 0 {
		t.Fatal("playlist list is empty")
	}

	// sometimes the playlist will only have 1 song and 1 empty track item
	if len(watchPlaylist) < 3 {
		t.Fatal("len(watchPlaylist) < 3")
	}

	for _, track := range watchPlaylist {
		if track.VideoID == "" {
			t.Fatal("track.VideoID == \"\"")
		}
		if track.Title == "" {
			t.Fatal("track.Title == \"\"")
		}
	}
}
