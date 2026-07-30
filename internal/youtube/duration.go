package youtube

import (
	"strconv"
	"strings"
	"time"
)

// parseISO8601Duration parses YouTube contentDetails.duration values like PT4M13S.
func parseISO8601Duration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" || s == "P0D" {
		return 0
	}
	if !strings.HasPrefix(s, "P") {
		return 0
	}
	s = strings.TrimPrefix(s, "P")
	var days, hours, mins, secs int
	if datePart, timePart, ok := strings.Cut(s, "T"); ok {
		days = takeDurationUnit(datePart, 'D')
		hours = takeDurationUnit(timePart, 'H')
		mins = takeDurationUnit(timePart, 'M')
		secs = takeDurationUnit(timePart, 'S')
	} else {
		days = takeDurationUnit(s, 'D')
	}
	return time.Duration(days)*24*time.Hour +
		time.Duration(hours)*time.Hour +
		time.Duration(mins)*time.Minute +
		time.Duration(secs)*time.Second
}

func takeDurationUnit(s string, unit byte) int {
	for i := range s {
		if s[i] == unit {
			// scan backward for digits
			j := i - 1
			for j >= 0 && s[j] >= '0' && s[j] <= '9' {
				j--
			}
			n, _ := strconv.Atoi(s[j+1 : i])
			return n
		}
	}
	return 0
}
