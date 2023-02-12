// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pubsub

import "time"

type Provider string

const (
	ProviderMemory Provider = "inmemory"
	ProviderRedis  Provider = "redis"
)

type Config struct {
	app       string // app namespace prefix
	namespace string

	provider Provider

	healthInterval time.Duration
	sendTimeout    time.Duration
	channelSize    int
}
