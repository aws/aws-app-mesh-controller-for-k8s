package utils

import "time"

const (
	PollIntervalShort  = 5 * time.Second
	PollIntervalMedium = 15 * time.Second
	PollRetries        = 10

	AWSPollIntervalShort  = 5 * time.Second
	AWSPollIntervalMedium = 15 * time.Second
)
