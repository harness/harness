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

package livelog

import (
	"context"
	"testing"
	"time"
)

func TestNewStream(t *testing.T) {
	s := newStream()
	if s == nil {
		t.Fatal("expected non-nil stream")
	}

	if s.list == nil {
		t.Error("expected non-nil subscriber list")
	}

	if len(s.list) != 0 {
		t.Errorf("expected empty subscriber list, got %d subscribers", len(s.list))
	}

	if len(s.hist) != 0 {
		t.Errorf("expected empty history, got %d lines", len(s.hist))
	}
}

func TestStream_Write(t *testing.T) {
	s := newStream()

	tests := []struct {
		name string
		line *Line
	}{
		{
			name: "basic line",
			line: &Line{
				Number:    1,
				Message:   "test message",
				Timestamp: time.Now().Unix(),
			},
		},
		{
			name: "empty message",
			line: &Line{
				Number:    2,
				Message:   "",
				Timestamp: time.Now().Unix(),
			},
		},
		{
			name: "zero timestamp",
			line: &Line{
				Number:    3,
				Message:   "zero timestamp",
				Timestamp: 0,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := s.write(test.line)
			if err != nil {
				t.Errorf("unexpected error writing line: %v", err)
			}
		})
	}

	// Verify all lines are in history
	if len(s.hist) != len(tests) {
		t.Errorf("expected %d lines in history, got %d", len(tests), len(s.hist))
	}
}

func TestStream_Write_BufferLimit(t *testing.T) {
	s := newStream()

	// Write more lines than buffer size
	numLines := bufferSize + 100
	for i := 0; i < numLines; i++ {
		line := &Line{
			Number:    i,
			Message:   "test message",
			Timestamp: int64(i),
		}
		err := s.write(line)
		if err != nil {
			t.Errorf("unexpected error writing line %d: %v", i, err)
		}
	}

	// History should be capped at buffer size
	if len(s.hist) != bufferSize {
		t.Errorf("expected history size %d, got %d", bufferSize, len(s.hist))
	}

	// Should contain the most recent lines
	firstLine := s.hist[0]
	expectedFirstNumber := numLines - bufferSize
	if firstLine.Number != expectedFirstNumber {
		t.Errorf("expected first line number %d, got %d", expectedFirstNumber, firstLine.Number)
	}

	lastLine := s.hist[len(s.hist)-1]
	expectedLastNumber := numLines - 1
	if lastLine.Number != expectedLastNumber {
		t.Errorf("expected last line number %d, got %d", expectedLastNumber, lastLine.Number)
	}
}

func TestStream_Subscribe(t *testing.T) {
	s := newStream()
	ctx := context.Background()

	// Subscribe to empty stream
	lineChan, errChan := s.subscribe(ctx)
	if lineChan == nil {
		t.Fatal("expected non-nil line channel")
	}
	if errChan == nil {
		t.Fatal("expected non-nil error channel")
	}

	// Verify subscriber was added
	if len(s.list) != 1 {
		t.Errorf("expected 1 subscriber, got %d", len(s.list))
	}
}

func TestStream_Subscribe_WithHistory(t *testing.T) {
	s := newStream()
	ctx := context.Background()

	// Write some lines to history
	historyLines := []*Line{
		{Number: 1, Message: "line 1", Timestamp: 1},
		{Number: 2, Message: "line 2", Timestamp: 2},
		{Number: 3, Message: "line 3", Timestamp: 3},
	}

	for _, line := range historyLines {
		err := s.write(line)
		if err != nil {
			t.Fatalf("failed to write line: %v", err)
		}
	}

	// Subscribe and receive history
	lineChan, _ := s.subscribe(ctx)

	// Should receive all historical lines
	for i, expectedLine := range historyLines {
		select {
		case receivedLine := <-lineChan:
			if receivedLine.Number != expectedLine.Number {
				t.Errorf("line %d: expected number %d, got %d", i, expectedLine.Number, receivedLine.Number)
			}
			if receivedLine.Message != expectedLine.Message {
				t.Errorf("line %d: expected message %q, got %q", i, expectedLine.Message, receivedLine.Message)
			}
		case <-time.After(time.Second):
			t.Errorf("timeout waiting for historical line %d", i)
		}
	}
}

func TestStream_Subscribe_NewLines(t *testing.T) {
	s := newStream()
	ctx := context.Background()

	// Subscribe first
	lineChan, _ := s.subscribe(ctx)

	// Write new lines
	newLines := []*Line{
		{Number: 1, Message: "new line 1", Timestamp: 1},
		{Number: 2, Message: "new line 2", Timestamp: 2},
	}

	for _, line := range newLines {
		err := s.write(line)
		if err != nil {
			t.Fatalf("failed to write line: %v", err)
		}

		// Should receive the new line
		select {
		case receivedLine := <-lineChan:
			if receivedLine.Number != line.Number {
				t.Errorf("expected number %d, got %d", line.Number, receivedLine.Number)
			}
			if receivedLine.Message != line.Message {
				t.Errorf("expected message %q, got %q", line.Message, receivedLine.Message)
			}
		case <-time.After(time.Second):
			t.Error("timeout waiting for new line")
		}
	}
}

func TestStream_Subscribe_ContextCancellation(t *testing.T) {
	s := newStream()
	ctx, cancel := context.WithCancel(context.Background())

	// Subscribe
	lineChan, errChan := s.subscribe(ctx)

	// Cancel context
	cancel()

	// Error channel should close
	select {
	case <-errChan:
		// Expected - error channel should close
	case <-time.After(time.Second):
		t.Error("timeout waiting for error channel to close")
	}

	// Line channel should also be closed eventually
	select {
	case _, ok := <-lineChan:
		if ok {
			t.Error("expected line channel to be closed")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for line channel to close")
	}
}

func TestStream_Close(t *testing.T) {
	s := newStream()
	ctx := context.Background()

	// Add some subscribers
	numSubscribers := 3
	for i := 0; i < numSubscribers; i++ {
		s.subscribe(ctx)
	}

	// Verify subscribers exist
	if len(s.list) != numSubscribers {
		t.Errorf("expected %d subscribers, got %d", numSubscribers, len(s.list))
	}

	// Close the stream
	err := s.close()
	if err != nil {
		t.Errorf("unexpected error closing stream: %v", err)
	}

	// All subscribers should be removed
	if len(s.list) != 0 {
		t.Errorf("expected 0 subscribers after close, got %d", len(s.list))
	}
}

func TestStream_MultipleSubscribers(t *testing.T) {
	s := newStream()
	ctx := context.Background()

	// Create multiple subscribers
	numSubscribers := 5
	channels := make([]<-chan *Line, numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		lineChan, _ := s.subscribe(ctx)
		channels[i] = lineChan
	}

	// Write a line
	line := &Line{
		Number:    1,
		Message:   "broadcast message",
		Timestamp: time.Now().Unix(),
	}

	err := s.write(line)
	if err != nil {
		t.Fatalf("failed to write line: %v", err)
	}

	// All subscribers should receive the line
	for i, ch := range channels {
		select {
		case receivedLine := <-ch:
			if receivedLine.Number != line.Number {
				t.Errorf("subscriber %d: expected number %d, got %d", i, line.Number, receivedLine.Number)
			}
			if receivedLine.Message != line.Message {
				t.Errorf("subscriber %d: expected message %q, got %q", i, line.Message, receivedLine.Message)
			}
		case <-time.After(time.Second):
			t.Errorf("subscriber %d: timeout waiting for line", i)
		}
	}
}

func TestBufferSize(t *testing.T) {
	expectedSize := 5000
	if bufferSize != expectedSize {
		t.Errorf("expected buffer size %d, got %d", expectedSize, bufferSize)
	}
}
