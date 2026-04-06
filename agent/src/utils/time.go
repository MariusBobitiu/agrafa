package utils

import "time"

const maxInt32 = int64(^uint32(0) >> 1)

func NowUTC() time.Time {
	return time.Now().UTC()
}

func DurationMillisecondsInt32(duration time.Duration) *int32 {
	milliseconds := duration.Milliseconds()
	if milliseconds > maxInt32 {
		milliseconds = maxInt32
	}

	value := int32(milliseconds)
	return &value
}
