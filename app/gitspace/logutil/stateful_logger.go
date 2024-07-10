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

package logutil

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/livelog"
)

const offset int64 = 1000000000

// StatefulLogger is a wrapper on livelog.Logstream. It is used to create stateful instances of LogStreamInstance.
type StatefulLogger struct {
	logz livelog.LogStream
}

// LogStreamInstance is a stateful instance of the livelog.LogStream. It keeps track of the position & log key (id).
type LogStreamInstance struct {
	ctx      context.Context
	id       int64
	offsetID int64
	position int
	scanner  *scanner
	logz     livelog.LogStream
}

func NewStatefulLogger(logz livelog.LogStream) *StatefulLogger {
	return &StatefulLogger{
		logz: logz,
	}
}

// GetLogStream returns an instance of LogStreamInstance tied to the given id.
func (s *StatefulLogger) CreateLogStream(ctx context.Context, id int64) (*LogStreamInstance, error) {
	// TODO: As livelog.LogStreamInstance uses only a single id as key, conflicts are likely if pipelines and gitspaces
	// are used in the same instance of Gitness. We need to update the underlying implementation to use another unique
	// key. To avoid that, we offset the ID by offset (1000000000).
	offsetID := offset + id

	// Create new logstream
	err := s.logz.Create(ctx, offsetID)
	if err != nil {
		return nil, fmt.Errorf("error creating log stream for ID %d: %w", id, err)
	}

	newStream := &LogStreamInstance{
		id:       id,
		offsetID: offsetID,
		ctx:      ctx,
		scanner:  newScanner(),
		logz:     s.logz,
	}

	return newStream, nil
}

// TailLogStream tails the underlying livelog.LogStream stream and returns the data and error channels.
func (s *StatefulLogger) TailLogStream(
	ctx context.Context,
	id int64,
) (<-chan *livelog.Line, <-chan error) {
	offsetID := offset + id
	return s.logz.Tail(ctx, offsetID)
}

// Write writes the msg into the underlying log stream.
func (l *LogStreamInstance) Write(msg string) error {
	lines, err := l.scanner.scan(msg)
	if err != nil {
		return fmt.Errorf("error parsing log lines %s: %w", msg, err)
	}

	now := time.Now().UnixMilli()

	for _, line := range lines {
		err = l.logz.Write(
			l.ctx,
			l.offsetID,
			&livelog.Line{
				Number:    l.position,
				Message:   line,
				Timestamp: now,
			})
		if err != nil {
			return fmt.Errorf("could not write log %s for ID %d at pos %d: %w", line, l.id, l.position, err)
		}

		l.position++
	}

	return nil
}

// Flush deletes the underlying stream.
func (l *LogStreamInstance) Flush() error {
	err := l.logz.Delete(l.ctx, l.offsetID)
	if err != nil {
		return fmt.Errorf("failed to delete old log stream for ID %d: %w", l.id, err)
	}

	return nil
}
