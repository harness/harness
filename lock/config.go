// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import "time"

type Provider string

const (
	MemoryProvider Provider = "inmemory"
	RedisProvider  Provider = "redis"
)

// A DelayFunc is used to decide the amount of time to wait between retries.
type DelayFunc func(tries int) time.Duration

type Config struct {
	app       string // app namespace prefix
	namespace string
	provider  Provider
	expiry    time.Duration

	tries      int
	retryDelay time.Duration
	delayFunc  DelayFunc

	driftFactor   float64
	timeoutFactor float64

	genValueFunc func() (string, error)
	value        string
}
