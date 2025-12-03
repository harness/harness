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
	Mode                  Mode
	Namespace             string
	MaxStreamLength       int64
	ApproxMaxStreamLength bool
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
