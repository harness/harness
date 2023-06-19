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
	App       string // app namespace prefix
	Namespace string
	Provider  Provider
	Expiry    time.Duration

	Tries      int
	RetryDelay time.Duration
	DelayFunc  DelayFunc

	DriftFactor   float64
	TimeoutFactor float64

	GenValueFunc func() (string, error)
	Value        string
}
