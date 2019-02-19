// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"io"
)

// Line represents a line in the logs.
type Line struct {
	Number    int    `json:"pos"`
	Message   string `json:"out"`
	Timestamp int64  `json:"time"`
}

// LogStore persists build output to storage.
type LogStore interface {
	// Find returns a log stream from the datastore.
	Find(ctx context.Context, stage int64) (io.ReadCloser, error)

	// Create writes copies the log stream from Reader r to the datastore.
	Create(ctx context.Context, stage int64, r io.Reader) error

	// Update writes copies the log stream from Reader r to the datastore.
	Update(ctx context.Context, stage int64, r io.Reader) error

	// Delete purges the log stream from the datastore.
	Delete(ctx context.Context, stage int64) error
}

// LogStream manages a live stream of logs.
type LogStream interface {
	// Create creates the log stream for the step ID.
	Create(context.Context, int64) error

	// Delete deletes the log stream for the step ID.
	Delete(context.Context, int64) error

	// Writes writes to the log stream.
	Write(context.Context, int64, *Line) error

	// Tail tails the log stream.
	Tail(context.Context, int64) (<-chan *Line, <-chan error)

	// Info returns internal stream information.
	Info(context.Context) *LogStreamInfo
}

// LogStreamInfo provides internal stream information. This can
// be used to monitor the number of registered streams and
// subscribers.
type LogStreamInfo struct {
	// Streams is a key-value pair where the key is the step
	// identifier, and the value is the count of subscribers
	// streaming the logs.
	Streams map[int64]int `json:"streams"`
}
