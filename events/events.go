// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
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
	Mode                  Mode   `json:"mode"`
	Namespace             string `json:"namespace"`
	MaxStreamLength       int64  `json:"max_stream_length"`
	ApproxMaxStreamLength bool   `json:"approx_max_stream_length"`
}
