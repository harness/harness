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
	"testing"
	"time"
)

func TestSubscriber_Publish(t *testing.T) {
	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	line := &Line{
		Number:    1,
		Message:   "test message",
		Timestamp: time.Now().Unix(),
	}

	// Publish line
	sub.publish(line)

	// Should receive the line
	select {
	case receivedLine := <-sub.handler:
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
		t.Error("timeout waiting for published line")
	}
}

func TestSubscriber_Publish_Multiple(t *testing.T) {
	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	lines := []*Line{
		{Number: 1, Message: "line 1", Timestamp: 1},
		{Number: 2, Message: "line 2", Timestamp: 2},
		{Number: 3, Message: "line 3", Timestamp: 3},
	}

	// Publish all lines
	for _, line := range lines {
		sub.publish(line)
	}

	// Should receive all lines in order
	for i, expectedLine := range lines {
		select {
		case receivedLine := <-sub.handler:
			if receivedLine.Number != expectedLine.Number {
				t.Errorf("line %d: expected number %d, got %d", i, expectedLine.Number, receivedLine.Number)
			}
			if receivedLine.Message != expectedLine.Message {
				t.Errorf("line %d: expected message %q, got %q", i, expectedLine.Message, receivedLine.Message)
			}
		case <-time.After(time.Second):
			t.Errorf("timeout waiting for line %d", i)
		}
	}
}

func TestSubscriber_Publish_BufferFull(t *testing.T) {
	// Create subscriber with small buffer for testing
	sub := &subscriber{
		handler: make(chan *Line, 2), // Small buffer
		closec:  make(chan struct{}),
		closed:  false,
	}

	// Fill the buffer
	line1 := &Line{Number: 1, Message: "line 1", Timestamp: 1}
	line2 := &Line{Number: 2, Message: "line 2", Timestamp: 2}
	line3 := &Line{Number: 3, Message: "line 3", Timestamp: 3}

	sub.publish(line1)
	sub.publish(line2)

	// Buffer should be full now, third publish should not block
	// (it should be dropped due to default case in select)
	sub.publish(line3)

	// Should receive first two lines
	receivedLine1 := <-sub.handler
	if receivedLine1.Number != 1 {
		t.Errorf("expected first line number 1, got %d", receivedLine1.Number)
	}

	receivedLine2 := <-sub.handler
	if receivedLine2.Number != 2 {
		t.Errorf("expected second line number 2, got %d", receivedLine2.Number)
	}

	// Channel should be empty now (third line was dropped)
	select {
	case <-sub.handler:
		t.Error("unexpected line received (should have been dropped)")
	default:
		// Expected - no more lines
	}
}

func TestSubscriber_Close(t *testing.T) {
	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	// Close the subscriber
	sub.close()

	// Should be marked as closed
	if !sub.closed {
		t.Error("subscriber should be marked as closed")
	}

	// Channels should be closed
	select {
	case <-sub.closec:
		// Expected - close channel should be closed
	default:
		t.Error("close channel should be closed")
	}

	select {
	case _, ok := <-sub.handler:
		if ok {
			t.Error("handler channel should be closed")
		}
	default:
		t.Error("handler channel should be closed")
	}
}

func TestSubscriber_Close_Multiple(t *testing.T) {
	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	// Close multiple times should not panic
	sub.close()
	sub.close()
	sub.close()

	// Should still be marked as closed
	if !sub.closed {
		t.Error("subscriber should be marked as closed")
	}
}

func TestSubscriber_Publish_AfterClose(_ *testing.T) {
	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	// Close the subscriber
	sub.close()

	// Publishing after close should not panic (due to recover in publish)
	line := &Line{Number: 1, Message: "test", Timestamp: 1}
	sub.publish(line) // Should not panic
}

func TestSubscriber_Publish_ClosedChannel(_ *testing.T) {
	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	// Close the close channel manually to simulate the condition
	close(sub.closec)

	line := &Line{Number: 1, Message: "test", Timestamp: 1}

	// Should not block or panic when closec is closed
	sub.publish(line)
}

func TestSubscriber_InitialState(t *testing.T) {
	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	// Initial state checks
	if sub.closed {
		t.Error("subscriber should not be closed initially")
	}

	if sub.handler == nil {
		t.Error("handler channel should not be nil")
	}

	if sub.closec == nil {
		t.Error("close channel should not be nil")
	}

	// Channels should be open
	select {
	case <-sub.closec:
		t.Error("close channel should not be closed initially")
	default:
		// Expected
	}
}

func TestSubscriber_ConcurrentPublish(t *testing.T) {
	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	// Start multiple goroutines publishing concurrently
	numGoroutines := 10
	linesPerGoroutine := 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < linesPerGoroutine; j++ {
				line := &Line{
					Number:    id*linesPerGoroutine + j,
					Message:   "concurrent message",
					Timestamp: int64(j),
				}
				sub.publish(line)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Drain the channel and count received lines
	receivedCount := 0
	for {
		select {
		case <-sub.handler:
			receivedCount++
		default:
			goto done
		}
	}
done:

	// Should have received some lines (may not be all due to buffer limits)
	if receivedCount == 0 {
		t.Error("should have received at least some lines")
	}

	// Should not have received more than total sent
	totalSent := numGoroutines * linesPerGoroutine
	if receivedCount > totalSent {
		t.Errorf("received more lines (%d) than sent (%d)", receivedCount, totalSent)
	}
}
