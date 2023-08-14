// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package livelog

import "context"

// Line represents a line in the logs.
type Line struct {
	Number    int    `json:"pos"`
	Message   string `json:"out"`
	Timestamp int64  `json:"time"`
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
