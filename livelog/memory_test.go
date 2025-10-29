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
	"errors"
	"testing"
	"time"
)

func TestNewMemory(t *testing.T) {
	stream := NewMemory()
	if stream == nil {
		t.Fatal("expected non-nil stream")
	}

	// Verify it implements LogStream interface
	var _ = stream
}

func TestStreamer_Create(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	tests := []struct {
		name string
		id   int64
	}{
		{
			name: "positive id",
			id:   1,
		},
		{
			name: "zero id",
			id:   0,
		},
		{
			name: "negative id",
			id:   -1,
		},
		{
			name: "large id",
			id:   9223372036854775807,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := stream.Create(ctx, test.id)
			if err != nil {
				t.Errorf("unexpected error creating stream: %v", err)
			}
		})
	}
}

func TestStreamer_Create_Multiple(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Create multiple streams
	ids := []int64{1, 2, 3, 100, -5}
	for _, id := range ids {
		err := stream.Create(ctx, id)
		if err != nil {
			t.Errorf("unexpected error creating stream %d: %v", id, err)
		}
	}

	// Verify all streams exist by checking info
	info := stream.Info(ctx)
	if len(info.Streams) != len(ids) {
		t.Errorf("expected %d streams, got %d", len(ids), len(info.Streams))
	}

	for _, id := range ids {
		if _, exists := info.Streams[id]; !exists {
			t.Errorf("stream %d not found in info", id)
		}
	}
}

func TestStreamer_Delete(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Create a stream first
	id := int64(1)
	err := stream.Create(ctx, id)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Delete the stream
	err = stream.Delete(ctx, id)
	if err != nil {
		t.Errorf("unexpected error deleting stream: %v", err)
	}

	// Verify stream is deleted
	info := stream.Info(ctx)
	if _, exists := info.Streams[id]; exists {
		t.Error("stream should have been deleted")
	}
}

func TestStreamer_Delete_NotFound(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Try to delete non-existent stream
	err := stream.Delete(ctx, 999)
	if !errors.Is(err, ErrStreamNotFound) {
		t.Errorf("expected ErrStreamNotFound, got %v", err)
	}
}

func TestStreamer_Write(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Create a stream
	id := int64(1)
	err := stream.Create(ctx, id)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Write to the stream
	line := &Line{
		Number:    1,
		Message:   "test message",
		Timestamp: time.Now().Unix(),
	}

	err = stream.Write(ctx, id, line)
	if err != nil {
		t.Errorf("unexpected error writing to stream: %v", err)
	}
}

func TestStreamer_Write_NotFound(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Try to write to non-existent stream
	line := &Line{
		Number:    1,
		Message:   "test message",
		Timestamp: time.Now().Unix(),
	}

	err := stream.Write(ctx, 999, line)
	if !errors.Is(err, ErrStreamNotFound) {
		t.Errorf("expected ErrStreamNotFound, got %v", err)
	}
}

func TestStreamer_Tail(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Create a stream
	id := int64(1)
	err := stream.Create(ctx, id)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Start tailing
	lines, errs := stream.Tail(ctx, id)
	if lines == nil || errs == nil {
		t.Fatal("expected non-nil channels")
	}

	// Write a line
	line := &Line{
		Number:    1,
		Message:   "test message",
		Timestamp: time.Now().Unix(),
	}

	err = stream.Write(ctx, id, line)
	if err != nil {
		t.Fatalf("failed to write to stream: %v", err)
	}

	// Read the line
	select {
	case receivedLine := <-lines:
		if receivedLine.Number != line.Number {
			t.Errorf("expected number %d, got %d", line.Number, receivedLine.Number)
		}
		if receivedLine.Message != line.Message {
			t.Errorf("expected message %q, got %q", line.Message, receivedLine.Message)
		}
		if receivedLine.Timestamp != line.Timestamp {
			t.Errorf("expected timestamp %d, got %d", line.Timestamp, receivedLine.Timestamp)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for line")
	}
}

func TestStreamer_Tail_NotFound(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Try to tail non-existent stream
	lines, errs := stream.Tail(ctx, 999)
	if lines != nil || errs != nil {
		t.Error("expected nil channels for non-existent stream")
	}
}

func TestStreamer_Tail_History(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Create a stream
	id := int64(1)
	err := stream.Create(ctx, id)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Write some lines before tailing
	lines := []*Line{
		{Number: 1, Message: "line 1", Timestamp: 1},
		{Number: 2, Message: "line 2", Timestamp: 2},
		{Number: 3, Message: "line 3", Timestamp: 3},
	}

	for _, line := range lines {
		err = stream.Write(ctx, id, line)
		if err != nil {
			t.Fatalf("failed to write line: %v", err)
		}
	}

	// Start tailing
	lineChan, _ := stream.Tail(ctx, id)

	// Should receive all historical lines
	for i, expectedLine := range lines {
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

func TestStreamer_Info(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Initially should have no streams
	info := stream.Info(ctx)
	if len(info.Streams) != 0 {
		t.Errorf("expected 0 streams, got %d", len(info.Streams))
	}

	// Create some streams
	ids := []int64{1, 2, 3}
	for _, id := range ids {
		err := stream.Create(ctx, id)
		if err != nil {
			t.Fatalf("failed to create stream %d: %v", id, err)
		}
	}

	// Check info again
	info = stream.Info(ctx)
	if len(info.Streams) != len(ids) {
		t.Errorf("expected %d streams, got %d", len(ids), len(info.Streams))
	}

	for _, id := range ids {
		if count, exists := info.Streams[id]; !exists {
			t.Errorf("stream %d not found in info", id)
		} else if count != 0 {
			t.Errorf("expected 0 subscribers for stream %d, got %d", id, count)
		}
	}
}

func TestStreamer_ConcurrentAccess(t *testing.T) {
	stream := NewMemory()
	ctx := context.Background()

	// Create a stream
	id := int64(1)
	err := stream.Create(ctx, id)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Start multiple goroutines writing to the stream
	done := make(chan bool)
	numWriters := 10
	linesPerWriter := 100

	for i := range numWriters {
		go func(writerID int) {
			defer func() { done <- true }()
			for j := range linesPerWriter {
				line := &Line{
					Number:    writerID*linesPerWriter + j,
					Message:   "concurrent message",
					Timestamp: time.Now().Unix(),
				}
				err := stream.Write(ctx, id, line)
				if err != nil {
					t.Errorf("writer %d: failed to write line %d: %v", writerID, j, err)
				}
			}
		}(i)
	}

	// Wait for all writers to complete
	for range numWriters {
		<-done
	}

	// Verify stream still exists
	info := stream.Info(ctx)
	if _, exists := info.Streams[id]; !exists {
		t.Error("stream should still exist after concurrent writes")
	}
}

func TestErrStreamNotFound(t *testing.T) {
	expectedMsg := "stream: not found"
	if ErrStreamNotFound.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, ErrStreamNotFound.Error())
	}
}
