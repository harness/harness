package pubsub

import (
	"time"
)

type Opts struct {
	// Timeout sets the expiration date for the channel,
	// at which time it will be closed and transmission will
	// stop. A zero value for means the channel will not timeout.
	Timeout time.Duration

	// Record indicates the channel should record the channel
	// activity and playback the full history to subscribers.
	Record bool
}

var DefaultOpts = &Opts{
	Timeout: 0,
	Record:  false,
}

var ConsoleOpts = &Opts{
	Timeout: time.Minute * 120,
	Record:  true,
}
