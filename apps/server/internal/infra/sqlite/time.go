package sqlite

import "time"

func encodeTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func decodeTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
