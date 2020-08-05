package utils

import "time"

const (
	PollIntervalShort  = 2 * time.Second
	PollIntervalMedium = 10 * time.Second
	PollRetries        = 5

	AWSPollIntervalShort = 200 * time.Millisecond
)
