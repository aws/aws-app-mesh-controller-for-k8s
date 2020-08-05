package utils

import "time"

const (
	PollIntervalShort  = 2 * time.Second
	PollIntervalMedium = 10 * time.Second
	PollRetries        = 5

	AWSPollIntervalShort  = 1 * time.Second
	AWSPollIntervalMedium = 5 * time.Second
)
