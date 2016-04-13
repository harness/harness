package fscache

import "time"

// Reaper is used to control when streams expire from the cache.
// It is called once right after loading, and then it is run
// again after every Next() period of time.
type Reaper interface {
	// Returns the amount of time to wait before the next scheduled Reaping.
	Next() time.Duration

	// Given a key and the last r/w times of a file, return true
	// to remove the file from the cache, false to keep it.
	Reap(key string, lastRead, lastWrite time.Time) bool
}

// NewReaper returns a simple reaper which runs every "period"
// and reaps files which are older than "expiry".
func NewReaper(expiry, period time.Duration) Reaper {
	return &reaper{
		expiry: expiry,
		period: period,
	}
}

type reaper struct {
	period time.Duration
	expiry time.Duration
}

func (g *reaper) Next() time.Duration {
	return g.period
}

func (g *reaper) Reap(key string, lastRead, lastWrite time.Time) bool {
	return lastRead.Before(time.Now().Add(-g.expiry))
}
