package ytmusic

// searchKey is YouTube Music's public WEB_REMIX InnerTube client key (not a user secret).
const searchKey = "AIzaSyC9XL3ZjWddXya6X74dJoCTL-WEYFDNX30"

var defaultRequestHeader = map[string][]string{
	"Content-Type": {"application/json"},
	"Accept":       {"*/*"},
	"Origin":       {origin},
	"Referer":      {"https://music.youtube.com/"},
	"User-Agent": {
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	},
}
