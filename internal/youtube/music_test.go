package youtube

import "testing"

func TestLooksLikeMusic(t *testing.T) {
	cases := []struct {
		name string
		v    Video
		want bool
	}{
		{"category10", Video{CategoryID: CategoryMusic, Title: "Anything"}, true},
		{"topic", Video{ChannelTitle: "Daft Punk - Topic", Title: "One More Time"}, true},
		{"official audio", Video{Title: "Song (Official Audio)", ChannelTitle: "Label"}, true},
		{"provided", Video{Title: "Track", ChannelTitle: "X"}, false}, // description not on Video
		{"vlog", Video{CategoryID: "22", Title: "My Day Vlog", ChannelTitle: "Creator"}, false},
	}
	for _, tc := range cases {
		if got := LooksLikeMusic(tc.v); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
	if !looksLikeMusicFields("22", "Label", "x", "Provided to YouTube by Universal") {
		t.Fatal("description hint")
	}
}

func TestFilterMusic(t *testing.T) {
	in := []Video{
		{VideoID: "1", CategoryID: "22", Title: "Vlog"},
		{VideoID: "2", CategoryID: CategoryMusic, Title: "Song"},
		{VideoID: "3", ChannelTitle: "A - Topic", Title: "Hit"},
	}
	out := FilterMusic(in)
	if len(out) != 2 || out[0].VideoID != "2" || out[1].VideoID != "3" {
		t.Fatalf("%+v", out)
	}
}
