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
