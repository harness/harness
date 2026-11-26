// Copyright 2026 Harness, Inc.
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
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func infNewRedisConsumer(t *testing.T, fake *infRedisFake) *RedisConsumer {
	t.Helper()
	c, err := NewRedisConsumer(fake.client(), "ns", "grp", "cons")
	require.NoError(t, err)
	return c
}

func infNoopHandler(context.Context, string, map[string]any) error { return nil }

func infDrainErrors(c *RedisConsumer) []error {
	var errs []error
	for {
		select {
		case e, ok := <-c.Errors():
			if !ok {
				return errs
			}
			errs = append(errs, e)
		default:
			return errs
		}
	}
}

func infDrainInfos(c *RedisConsumer) []string {
	var infos []string
	for {
		select {
		case i, ok := <-c.Infos():
			if !ok {
				return infos
			}
			infos = append(infos, i)
		default:
			return infos
		}
	}
}

func infErrorsContain(errs []error, part string) bool {
	for _, e := range errs {
		if strings.Contains(e.Error(), part) {
			return true
		}
	}
	return false
}

func infInfosContain(infos []string, part string) bool {
	for _, i := range infos {
		if strings.Contains(i, part) {
			return true
		}
	}
	return false
}

func TestInfRedisConsumer_Validation(t *testing.T) {
	fake := infNewRedisFake(func([]string) any { return infRedisSimple("OK") })
	_, err := NewRedisConsumer(fake.client(), "ns", "", "c")
	require.Error(t, err)
	_, err = NewRedisConsumer(fake.client(), "ns", "g", "")
	require.Error(t, err)

	c := infNewRedisConsumer(t, fake)
	require.Error(t, c.Register("", infNoopHandler))
	require.Error(t, c.Register("s", nil))
	require.NoError(t, c.Register("s", infNoopHandler, WithMaxRetries(5)))
	require.Equal(t, 5, c.streams["ns:s"].config.maxRetries)
	require.ErrorContains(t, c.Register("s", infNoopHandler), "already registered")

	c.Configure(WithConcurrency(4))
	require.Equal(t, 4, c.Config.Concurrency)

	empty := infNewRedisConsumer(t, fake)
	require.EqualError(t, empty.Start(context.Background()), "no streams registered")
}

func TestInfRedisConsumer_StartErrors(t *testing.T) {
	t.Run("ping fails", func(t *testing.T) {
		fake := infNewRedisFake(func([]string) any { return infRedisErr("ERR down") })
		c := infNewRedisConsumer(t, fake)
		require.NoError(t, c.Register("s", infNoopHandler))
		require.ErrorContains(t, c.Start(context.Background()), "failed to ping redis server")
	})
	t.Run("group creation fails", func(t *testing.T) {
		fake := infNewRedisFake(func(args []string) any {
			if strings.EqualFold(args[0], "xgroup") {
				return infRedisErr("ERR nope")
			}
			return infRedisSimple("PONG")
		})
		c := infNewRedisConsumer(t, fake)
		require.NoError(t, c.Register("s", infNoopHandler))
		require.ErrorContains(t, c.Start(context.Background()), "failed to create consumer group 'grp'")
	})
}

func TestInfRedisConsumer_StartProcessesMessages(t *testing.T) {
	var delivered atomic.Bool
	fake := infNewRedisFake(func(args []string) any {
		switch strings.ToLower(args[0]) {
		case "ping":
			return infRedisSimple("PONG")
		case infCmdXGroup:
			return infRedisErr("BUSYGROUP Consumer Group name already exists")
		case "xinfo":
			return []any{}
		case infCmdXReadGroup:
			if args[len(args)-1] == ">" && delivered.CompareAndSwap(false, true) {
				return infStreamReply(infStreamMsg("1-0", "k", "v"))
			}
			if args[len(args)-1] == "0" {
				return infStreamReply()
			}
			return infSlowNil()
		default:
			return int64(1)
		}
	})
	c := infNewRedisConsumer(t, fake)
	c.Configure(WithConcurrency(2))

	var got atomic.Value
	require.NoError(t, c.Register("s", func(_ context.Context, id string, payload map[string]any) error {
		value, _ := payload["k"].(string)
		got.Store(id + ":" + value)
		return nil
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, c.Start(ctx))
	require.ErrorIs(t, c.Start(ctx), ErrAlreadyStarted)
	require.ErrorIs(t, c.Register("x", infNoopHandler), ErrAlreadyStarted)
	c.Configure(WithConcurrency(9))
	require.Equal(t, 2, c.Config.Concurrency)

	require.Eventually(t, func() bool { return got.Load() == "1-0:v" }, 5*time.Second, 5*time.Millisecond)
	require.Eventually(t, func() bool { return fake.count("xack") == 1 }, 5*time.Second, 5*time.Millisecond)

	cancel()
	require.Eventually(t, func() bool {
		_, ok := <-c.Errors()
		return !ok
	}, 5*time.Second, 5*time.Millisecond)
}

func TestInfRedisConsumer_ReaderHistoryThenNewMessages(t *testing.T) {
	var calls atomic.Int32
	fake := infNewRedisFake(func(args []string) any {
		if !strings.EqualFold(args[0], "xreadgroup") {
			return infRedisSimple("OK")
		}
		switch calls.Add(1) {
		case 1:
			require.Equal(t, "0", args[len(args)-1])
			return infStreamReply(infStreamMsg("1-0", "a", "1"))
		case 2:
			require.Equal(t, "1-0", args[len(args)-1])
			return infStreamReply()
		case 3:
			require.Equal(t, ">", args[len(args)-1])
			return infStreamReply(infStreamMsg("2-0", "b", "2"), infStreamMsg("3-0", "c", "3"))
		default:
			return infSlowNil()
		}
	})
	c := infNewRedisConsumer(t, fake)
	require.NoError(t, c.Register("s", infNoopHandler))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { c.reader(ctx); close(done) }()

	var ids []string
	for len(ids) < 3 {
		select {
		case m := <-c.messageQueue:
			require.Equal(t, "ns:s", m.streamID)
			ids = append(ids, m.id)
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out; got %v", ids)
		}
	}
	require.Equal(t, []string{"1-0", "2-0", "3-0"}, ids)
	cancel()
	<-done

	infos := infDrainInfos(c)
	require.True(t, infInfosContain(infos, "had 1 more messages in the history"))
	require.True(t, infInfosContain(infos, "completed scan of history"))
}

func TestInfRedisConsumer_ReaderErrorHandling(t *testing.T) {
	var calls atomic.Int32
	fake := infNewRedisFake(func(args []string) any {
		switch strings.ToLower(args[0]) {
		case infCmdXGroup:
			return infRedisSimple("OK")
		case infCmdXReadGroup:
			switch calls.Add(1) {
			case 1:
				return infRedisErr("NOGROUP No such key or consumer group")
			case 2:
				return infRedisErr("ERR generic failure")
			default:
				return infSlowNil()
			}
		}
		return infRedisSimple("OK")
	})
	c := infNewRedisConsumer(t, fake)
	require.NoError(t, c.Register("s", infNoopHandler))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { c.reader(ctx); close(done) }()

	// first failure re-creates the group, second one is reported with a 5s backoff afterwards.
	require.Eventually(t, func() bool { return fake.count("xgroup") >= 1 && calls.Load() >= 2 },
		5*time.Second, 5*time.Millisecond)
	cancel()
	<-done

	require.True(t, infInfosContain(infDrainInfos(c), "re-created group for all streams"))
	require.True(t, infErrorsContain(infDrainErrors(c), "failed to read redis streams"))
}

func TestInfRedisConsumer_ReaderNoGroupRecreateFails(t *testing.T) {
	fake := infNewRedisFake(func(args []string) any {
		switch strings.ToLower(args[0]) {
		case infCmdXGroup:
			return infRedisErr("ERR cannot create")
		case infCmdXReadGroup:
			return infRedisErr("NOGROUP missing")
		}
		return infRedisSimple("OK")
	})
	c := infNewRedisConsumer(t, fake)
	require.NoError(t, c.Register("s", infNoopHandler))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { c.reader(ctx); close(done) }()

	require.Eventually(t, func() bool { return fake.count("xgroup") >= 1 }, 5*time.Second, 5*time.Millisecond)
	cancel()
	<-done
	require.True(t, infErrorsContain(infDrainErrors(c), "failed to re-create group"))
}

func TestInfRedisConsumer_ConsumerLoop(t *testing.T) {
	var acked sync.Map
	fake := infNewRedisFake(func(args []string) any {
		if strings.EqualFold(args[0], "xack") {
			if args[len(args)-1] == "bad-ack" {
				return infRedisErr("ERR ack failed")
			}
			acked.Store(args[len(args)-1], true)
		}
		return int64(1)
	})
	c := infNewRedisConsumer(t, fake)

	var handled sync.Map
	require.NoError(t, c.Register("s", func(_ context.Context, id string, _ map[string]any) error {
		handled.Store(id, true)
		switch id {
		case "fail":
			return errors.New("handler failed")
		case "panic":
			panic("oops")
		}
		return nil
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { c.consumer(ctx); close(done) }()

	for _, id := range []string{"ok", "fail", "panic", "bad-ack"} {
		c.messageQueue <- message{streamID: "ns:s", id: id}
	}
	c.messageQueue <- message{streamID: "ns:other", id: "stray"}
	c.messageQueue <- message{} // empty id terminates the consumer
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer did not stop on empty message id")
	}

	_, ok := acked.Load("ok")
	require.True(t, ok)
	_, ok = acked.Load("fail")
	require.False(t, ok, "failed messages must not be acknowledged")
	_, ok = handled.Load("stray")
	require.False(t, ok)

	errs := infDrainErrors(c)
	require.True(t, infErrorsContain(errs, "failed to process message 'fail'"))
	require.True(t, infErrorsContain(errs, "PANIC when processing message 'panic'"))
	require.True(t, infErrorsContain(errs, "failed to acknowledge message 'bad-ack'"))
	require.True(t, infErrorsContain(errs, "message 'stray' in stream 'ns:other' that doesn't belong to us"))
}

func TestInfRedisConsumer_ConsumerStopsOnContextCancel(t *testing.T) {
	c := infNewRedisConsumer(t, infNewRedisFake(func([]string) any { return int64(1) }))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { c.consumer(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer did not stop")
	}
}

func infPendingEntry(id string, retries int64) []any {
	return []any{id, "other-consumer", int64(120000), retries}
}

func TestInfRedisConsumer_Reclaimer(t *testing.T) {
	var pendingCalls atomic.Int32
	fake := infNewRedisFake(func(args []string) any {
		switch strings.ToLower(args[0]) {
		case "xpending":
			switch pendingCalls.Add(1) {
			case 1:
				return infRedisErr("ERR pending failed")
			case 2:
				return []any{
					infPendingEntry("1-0", 9), // discarded: acked fine
					infPendingEntry("2-0", 9), // discarded: ack fails
					infPendingEntry("3-0", 1), // claim returns nil: acked
					infPendingEntry("4-0", 1), // claim error
					infPendingEntry("5-0", 1), // claim returns nothing
					infPendingEntry("6-0", 1), // claimed and enqueued
					infPendingEntry("7-0", 1), // claim nil and ack fails
				}
			default:
				return []any{}
			}
		case "xack":
			id := args[len(args)-1]
			if id == "2-0" || id == "7-0" {
				return infRedisErr("ERR ack failed")
			}
			return int64(1)
		case "xclaim":
			id := args[len(args)-1]
			switch id {
			case "3-0", "7-0":
				return nil
			case "4-0":
				return infRedisErr("ERR claim failed")
			case "5-0":
				return []any{}
			default:
				return []any{infStreamMsg(id, "k", "v")}
			}
		}
		return infRedisSimple("OK")
	})
	c := infNewRedisConsumer(t, fake)
	require.NoError(t, c.Register("s", infNoopHandler))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { c.reclaimer(ctx, 5*time.Millisecond); close(done) }()

	select {
	case m := <-c.messageQueue:
		require.Equal(t, "6-0", m.id)
		require.Equal(t, "ns:s", m.streamID)
		require.Equal(t, "v", m.values["k"])
	case <-time.After(5 * time.Second):
		t.Fatal("expected a reclaimed message")
	}
	// wait for the next poll so the whole batch has been processed
	require.Eventually(t, func() bool { return pendingCalls.Load() >= 3 }, 5*time.Second, 5*time.Millisecond)
	cancel()
	<-done

	errs := infDrainErrors(c)
	require.True(t, infErrorsContain(errs, "failed to fetch pending messages"))
	require.True(t, infErrorsContain(errs, "failed to claim message '4-0'"))
	require.True(t, infErrorsContain(errs, "redis returned no message"))
	require.True(t, infErrorsContain(errs, "failed to acknowledge failed message '7-0'"))

	var discarded []*DiscardedMessageError
	for _, e := range errs {
		var d *DiscardedMessageError
		if errors.As(e, &d) {
			discarded = append(discarded, d)
		}
	}
	require.Len(t, discarded, 2)
	require.Equal(t, "1-0", discarded[0].MessageID)
	require.EqualValues(t, 8, discarded[0].RetryCount) // redis counts the current delivery
	require.NoError(t, discarded[0].AckErr)
	require.Equal(t, "2-0", discarded[1].MessageID)
	require.Error(t, discarded[1].AckErr)

	require.True(t, infInfosContain(infDrainInfos(c), "acknowledged failed message '3-0'"))
}

func TestInfRedisConsumer_ReclaimerStopsOnCancel(t *testing.T) {
	c := infNewRedisConsumer(t, infNewRedisFake(func([]string) any { return []any{} }))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { c.reclaimer(ctx, time.Hour); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reclaimer did not stop")
	}
}

func TestInfRedisConsumer_RemoveStaleConsumers(t *testing.T) {
	consumerEntry := func(name string, pending, idleMs int64) []any {
		return []any{"name", name, "pending", pending, "idle", idleMs, "inactive", idleMs}
	}
	day := int64(24 * time.Hour / time.Millisecond)

	t.Run("deletes only old idle consumers", func(t *testing.T) {
		fake := infNewRedisFake(func(args []string) any {
			switch strings.ToLower(args[0]) {
			case "xinfo":
				return []any{
					consumerEntry("stale", 0, day),
					consumerEntry("stale-pending", 3, day),
					consumerEntry("fresh", 0, 1000),
					consumerEntry("stale-delfail", 0, day),
				}
			case infCmdXGroup:
				if args[len(args)-1] == "stale-delfail" {
					return infRedisErr("ERR delete failed")
				}
				return int64(0)
			}
			return infRedisSimple("OK")
		})
		c := infNewRedisConsumer(t, fake)
		require.NoError(t, c.Register("s", infNoopHandler))
		c.removeStaleConsumers(context.Background(), time.Hour)

		require.Equal(t, 2, fake.count("xgroup", "delconsumer"))
		require.True(t, infInfosContain(infDrainInfos(c), "removed stale consumer 'stale'"))
		require.True(t, infErrorsContain(infDrainErrors(c), "failed to remove stale consumer 'stale-delfail'"))
	})

	t.Run("read failure is reported", func(t *testing.T) {
		fake := infNewRedisFake(func([]string) any { return infRedisErr("ERR boom") })
		c := infNewRedisConsumer(t, fake)
		require.NoError(t, c.Register("s", infNoopHandler))
		c.removeStaleConsumers(context.Background(), time.Hour)
		require.True(t, infErrorsContain(infDrainErrors(c), "failed to read consumers for stream 'ns:s'"))
	})
}

func TestInfCreateGroup(t *testing.T) {
	ctx := context.Background()

	fake := infNewRedisFake(func([]string) any { return infRedisSimple("OK") })
	require.NoError(t, createGroup(ctx, fake.client(), "ns:s", "grp"))
	cmd := fake.commands()[0]
	require.Equal(t, []string{"xgroup", "create", "ns:s", "grp", "$", "mkstream"}, cmd)

	busy := infNewRedisFake(func([]string) any { return infRedisErr("BUSYGROUP Consumer Group name already exists") })
	require.NoError(t, createGroup(ctx, busy.client(), "ns:s", "grp"))

	bad := infNewRedisFake(func([]string) any { return infRedisErr("ERR boom") })
	require.ErrorContains(t, createGroup(ctx, bad.client(), "ns:s", "grp"), "failed to create consumer group")
}

func TestInfRedisConsumer_PushHelpersDoNotBlock(t *testing.T) {
	c := infNewRedisConsumer(t, infNewRedisFake(func([]string) any { return nil }))
	for i := 0; i < cap(c.errorCh)+5; i++ {
		c.pushError(errors.New("x"))
	}
	for i := 0; i < cap(c.infoCh)+5; i++ {
		c.pushInfo("x")
	}
	require.Len(t, c.errorCh, cap(c.errorCh))
	require.Len(t, c.infoCh, cap(c.infoCh))
}
