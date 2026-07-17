package ytmusic

import (
	"math"
	"strconv"
	"strings"
)

// durationToInt converts the duration string ("4:20") to seconds (260).
func durationToInt(duration any) int {
	s, ok := duration.(string)
	if !ok || s == "" || !strings.Contains(s, ":") {
		return 0
	}
	items := strings.Split(s, ":")
	result := 0
	for i := range items {
		durationInt, err := strconv.Atoi(items[i])
		if err != nil {
			return 0
		}
		result += durationInt * int(math.Pow(60, float64(len(items)-i-1)))
	}
	return result
}
