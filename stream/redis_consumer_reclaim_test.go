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

package stream

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

// TestReclaimerReclaimsOlderPendingMessage checks that a pending message is reclaimed even when a newer
// one was reclaimed before it became idle. Message 1 is being retried (its idle time is fresh) while
// message 2 is idle, so the first pass only reclaims message 2. Once message 1 is idle too, the next pass
// has to reclaim it: the pending list must be scanned from its start, not from the last message seen.
func TestReclaimerReclaimsOlderPendingMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mr := miniredis.RunT(t)
	now := time.Now()
	mr.SetTime(now)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	const (
		group       = "group"
		idleTimeout = MinIdleTimeout
	)
	c, err := NewRedisConsumer(rdb, "ns", group, "consumer")
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Register("s", func(context.Context, string, map[string]any) error { return nil },
		WithIdleTimeout(idleTimeout), WithMaxRetries(5)); err != nil {
		t.Fatal(err)
	}
	if err = c.createGroupForAllStreams(ctx); err != nil {
		t.Fatal(err)
	}
	streamID := transposeStreamID("ns", "s")

	// Two messages are delivered to another consumer, which never acks them.
	for _, id := range []string{"1-0", "2-0"} {
		if err = rdb.XAdd(ctx, &redis.XAddArgs{Stream: streamID, ID: id, Values: []string{"k", id}}).Err(); err != nil {
			t.Fatal(err)
		}
	}
	if err = rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: group, Consumer: "other", Streams: []string{streamID, ">"}, Count: 2,
	}).Err(); err != nil {
		t.Fatal(err)
	}

	// Both are idle now, but message 1 is being retried, which refreshes its idle time.
	now = now.Add(2 * idleTimeout)
	mr.SetTime(now)
	if err = rdb.XClaim(ctx, &redis.XClaimArgs{
		Stream: streamID, Group: group, Consumer: "other", Messages: []string{"1-0"},
	}).Err(); err != nil {
		t.Fatal(err)
	}

	go c.reclaimer(ctx, 10*time.Millisecond)

	// The first pass reclaims only message 2.
	if id := nextReclaimed(t, c); id != "2-0" {
		t.Fatalf("first reclaimed message %q, want 2-0", id)
	}

	// Message 1's retry failed too, so it is idle again: it has to be reclaimed.
	mr.SetTime(now.Add(2 * idleTimeout))
	for {
		if id := nextReclaimed(t, c); id == "1-0" {
			return
		}
	}
}

// nextReclaimed returns the ID of the next message the reclaimer queues for processing.
func nextReclaimed(t *testing.T, c *RedisConsumer) string {
	t.Helper()
	select {
	case m := <-c.messageQueue:
		return m.id
	case <-time.After(2 * time.Second):
		t.Fatal("no message was reclaimed")
		return ""
	}
}
