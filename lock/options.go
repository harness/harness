// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import (
	"time"
)

// An Option configures a mutex.
type Option interface {
	Apply(*Config)
}

// OptionFunc is a function that configures a mutex.
type OptionFunc func(*Config)

// Apply calls f(config).
func (f OptionFunc) Apply(config *Config) {
	f(config)
}

// WithNamespace returns an option that configures Mutex.ns.
func WithNamespace(ns string) Option {
	return OptionFunc(func(m *Config) {
		m.namespace = ns
	})
}

// WithExpiry can be used to set the expiry of a mutex to the given value.
func WithExpiry(expiry time.Duration) Option {
	return OptionFunc(func(m *Config) {
		m.expiry = expiry
	})
}

// WithTries can be used to set the number of times lock acquire is attempted.
func WithTries(tries int) Option {
	return OptionFunc(func(m *Config) {
		m.tries = tries
	})
}

// WithRetryDelay can be used to set the amount of time to wait between retries.
func WithRetryDelay(delay time.Duration) Option {
	return OptionFunc(func(m *Config) {
		m.delayFunc = func(tries int) time.Duration {
			return delay
		}
	})
}

// WithRetryDelayFunc can be used to override default delay behavior.
func WithRetryDelayFunc(delayFunc DelayFunc) Option {
	return OptionFunc(func(m *Config) {
		m.delayFunc = delayFunc
	})
}

// WithDriftFactor can be used to set the clock drift factor.
func WithDriftFactor(factor float64) Option {
	return OptionFunc(func(m *Config) {
		m.driftFactor = factor
	})
}

// WithTimeoutFactor can be used to set the timeout factor.
func WithTimeoutFactor(factor float64) Option {
	return OptionFunc(func(m *Config) {
		m.timeoutFactor = factor
	})
}

// WithGenValueFunc can be used to set the custom value generator.
func WithGenValueFunc(genValueFunc func() (string, error)) Option {
	return OptionFunc(func(m *Config) {
		m.genValueFunc = genValueFunc
	})
}

// WithValue can be used to assign the random value without having to call lock.
// This allows the ownership of a lock to be "transferred" and allows the lock to be unlocked from elsewhere.
func WithValue(v string) Option {
	return OptionFunc(func(m *Config) {
		m.value = v
	})
}
