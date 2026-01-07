// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		m.Namespace = ns
	})
}

// WithExpiry can be used to set the expiry of a mutex to the given value.
func WithExpiry(expiry time.Duration) Option {
	return OptionFunc(func(m *Config) {
		m.Expiry = expiry
	})
}

// WithTries can be used to set the number of times lock acquire is attempted.
func WithTries(tries int) Option {
	return OptionFunc(func(m *Config) {
		m.Tries = tries
	})
}

// WithRetryDelay can be used to set the amount of time to wait between retries.
func WithRetryDelay(delay time.Duration) Option {
	return OptionFunc(func(m *Config) {
		m.DelayFunc = func(_ int) time.Duration {
			return delay
		}
	})
}

// WithRetryDelayFunc can be used to override default delay behavior.
func WithRetryDelayFunc(delayFunc DelayFunc) Option {
	return OptionFunc(func(m *Config) {
		m.DelayFunc = delayFunc
	})
}

// WithDriftFactor can be used to set the clock drift factor.
func WithDriftFactor(factor float64) Option {
	return OptionFunc(func(m *Config) {
		m.DriftFactor = factor
	})
}

// WithTimeoutFactor can be used to set the timeout factor.
func WithTimeoutFactor(factor float64) Option {
	return OptionFunc(func(m *Config) {
		m.TimeoutFactor = factor
	})
}

// WithGenValueFunc can be used to set the custom value generator.
func WithGenValueFunc(genValueFunc func() (string, error)) Option {
	return OptionFunc(func(m *Config) {
		m.GenValueFunc = genValueFunc
	})
}

// WithValue can be used to assign the random value without having to call lock.
// This allows the ownership of a lock to be "transferred" and allows the lock to be unlocked from elsewhere.
func WithValue(v string) Option {
	return OptionFunc(func(m *Config) {
		m.Value = v
	})
}
