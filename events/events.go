// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"errors"
	"fmt"
	"time"
)

const (
	// streamPayloadKey is the key used for storing the event in a stream message.
	streamPayloadKey = "event"
)

type Event[T interface{}] struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Payload   T         `json:"payload"`
}

// EventType describes the type of event.
type EventType string

// getStreamID generates the streamID for a given category and type of event.
func getStreamID(category string, event EventType) string {
	return fmt.Sprintf("events:%s:%s", category, event)
}

// Mode defines the different modes of the event framework.
type Mode string

const (
	ModeRedis    Mode = "redis"
	ModeInMemory Mode = "inmemory"
)

// Config defines the config of the events system.
type Config struct {
	Mode                  Mode   `envconfig:"GITNESS_EVENTS_MODE"                     default:"inmemory"`
	Namespace             string `envconfig:"GITNESS_EVENTS_NAMESPACE"                default:"gitness"`
	MaxStreamLength       int64  `envconfig:"GITNESS_EVENTS_MAX_STREAM_LENGTH"        default:"10000"`
	ApproxMaxStreamLength bool   `envconfig:"GITNESS_EVENTS_APPROX_MAX_STREAM_LENGTH" default:"true"`
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}
	if c.Mode != ModeRedis && c.Mode != ModeInMemory {
		return fmt.Errorf("config.Mode '%s' is not supported", c.Mode)
	}
	if c.MaxStreamLength < 1 {
		return errors.New("config.MaxStreamLength has to be a positive number")
	}

	return nil
}
