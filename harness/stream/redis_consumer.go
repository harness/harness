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
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisConsumer provides functionality to process Redis streams as part of a consumer group.
type RedisConsumer struct {
	rdb redis.UniversalClient
	// namespace specifies the namespace of the keys - any stream key will be prefixed with it
	namespace string
	// groupName specifies the name of the consumer group.
	groupName string
	// consumerName specifies the name of the consumer.
	consumerName string

	// Config is the generic consumer configuration.
	Config ConsumerConfig

	// streams is a map of all registered streams and their handlers.
	streams map[string]handler

	isStarted    bool
	messageQueue chan message
	errorCh      chan error
	infoCh       chan string
}

// NewRedisConsumer creates new Redis stream consumer. Streams are read with XREADGROUP.
// It returns channels of info messages and errors. The caller should not block on these channels for too long.
// These channels are provided mainly for logging.
func NewRedisConsumer(rdb redis.UniversalClient, namespace string,
	groupName string, consumerName string) (*RedisConsumer, error) {
	if groupName == "" {
		return nil, errors.New("groupName can't be empty")
	}
	if consumerName == "" {
		return nil, errors.New("consumerName can't be empty")
	}

	const queueCapacity = 500
	const errorChCapacity = 64
	const infoChCapacity = 64

	return &RedisConsumer{
		rdb:          rdb,
		namespace:    namespace,
		groupName:    groupName,
		consumerName: consumerName,
		streams:      map[string]handler{},
		Config:       defaultConfig,
		isStarted:    false,
		messageQueue: make(chan message, queueCapacity),
		errorCh:      make(chan error, errorChCapacity),
		infoCh:       make(chan string, infoChCapacity),
	}, nil
}

func (c *RedisConsumer) Configure(opts ...ConsumerOption) {
	if c.isStarted {
		return
	}

	for _, opt := range opts {
		opt.apply(&c.Config)
	}
}

func (c *RedisConsumer) Register(streamID string, fn HandlerFunc, opts ...HandlerOption) error {
	if c.isStarted {
		return ErrAlreadyStarted
	}
	if streamID == "" {
		return errors.New("streamID can't be empty")
	}
	if fn == nil {
		return errors.New("fn can't be empty")
	}

	// transpose streamID to key namespace - no need to keep inner streamID
	transposedStreamID := transposeStreamID(c.namespace, streamID)
	if _, ok := c.streams[transposedStreamID]; ok {
		return fmt.Errorf("consumer is already registered for '%s' (redis stream '%s')", streamID, transposedStreamID)
	}

	// create final config for handler
	config := c.Config.DefaultHandlerConfig
	for _, opt := range opts {
		opt.apply(&config)
	}

	c.streams[transposedStreamID] = handler{
		handle: fn,
		config: config,
	}

	return nil
}

func (c *RedisConsumer) Start(ctx context.Context) error {
	if c.isStarted {
		return ErrAlreadyStarted
	}

	if len(c.streams) == 0 {
		return errors.New("no streams registered")
	}

	var err error

	// Check if Redis is accessible, fail if it's not.
	err = c.rdb.Ping(ctx).Err()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("failed to ping redis server: %w", err)
	}

	// Create consumer group for all streams, creates streams if they don't exist.
	err = c.createGroupForAllStreams(ctx)
	if err != nil {
		return err
	}

	// mark as started before starting go routines (can't error out from here)
	c.isStarted = true

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.removeStaleConsumers(ctx, time.Hour)
		// launch redis reader, it will finish when the ctx is done
		c.reader(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		// launch redis message reclaimer, it will finish when the ctx is done.
		// IMPORTANT: Keep reclaim interval small for now to support faster retries => higher load on redis!
		// TODO: Make retries local by default with opt-in cross-instance retries.
		// https://harness.atlassian.net/browse/SCM-83
		const reclaimInterval = 10 * time.Second
		c.reclaimer(ctx, reclaimInterval)
	}()

	for i := 0; i < c.Config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// launch redis message consumer, it will finish when the ctx is done
			c.consumer(ctx)
		}()
	}

	go func() {
		// wait for all go routines to complete
		wg.Wait()

		// close all channels
		close(c.messageQueue)
		close(c.errorCh)
		close(c.infoCh)
	}()

	return nil
}

// reader method reads a Redis stream with XREADGROUP command to retrieve messages.
// The messages are then sent to a go channel passed as parameter for processing.
// If the stream already contains unassigned messages, those we'll be returned.
// Otherwise XREADGROUP blocks until either a new message arrives or block timeout happens.
// The method terminates when the provided context finishes.
//
//nolint:funlen,gocognit // refactor if needed
func (c *RedisConsumer) reader(ctx context.Context) {
	delays := []time.Duration{1 * time.Millisecond, 5 * time.Second, 15 * time.Second, 30 * time.Second, time.Minute}
	consecutiveFailures := 0

	// pre-generate streams argument for XReadGroup
	// NOTE: for the first call ever we want to get the history of the consumer (to allow for seamless restarts)
	// ASSUMPTION: only one consumer with a given groupName+consumerName is running at a time
	scanHistory := true
	streamLen := len(c.streams)
	streamsArg := make([]string, 2*streamLen)
	i := 0
	for streamID := range c.streams {
		streamsArg[i] = streamID
		streamsArg[streamLen+i] = "0"
		i++
	}

	for {
		var delay time.Duration
		if consecutiveFailures < len(delays) {
			delay = delays[consecutiveFailures]
		} else {
			delay = delays[len(delays)-1]
		}
		readTimer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			readTimer.Stop()
			return

		case <-readTimer.C:
			const count = 100

			resReadStream, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    c.groupName,
				Consumer: c.consumerName,
				Streams:  streamsArg,
				Count:    count,
				Block:    5 * time.Minute,
			}).Result()

			// if context is canceled, continue and next iteration will exit cleanly
			if errors.Is(err, context.Canceled) {
				continue
			}

			// network timeout - log it and retry
			var errNet net.Error
			if ok := errors.As(err, &errNet); ok && errNet.Timeout() {
				c.pushError(fmt.Errorf("encountered network failure: %w", errNet))
				consecutiveFailures++
				continue
			}

			// group doesn't exist anymore - recreate it
			if err != nil && strings.HasPrefix(err.Error(), "NOGROUP") {
				cErr := c.createGroupForAllStreams(ctx)
				if cErr != nil {
					c.pushError(fmt.Errorf("failed to re-create group for at least one stream: %w", err))
					consecutiveFailures++
				} else {
					c.pushInfo(fmt.Sprintf("re-created group for all streams where it got removed, original error: %s",
						err))
					consecutiveFailures = 0
				}
				continue
			}

			// any other error we handle generically
			if err != nil && !errors.Is(err, redis.Nil) {
				consecutiveFailures++
				c.pushError(fmt.Errorf("failed to read redis streams %v (consecutive fails: %d): %w",
					streamsArg, consecutiveFailures, err))
				continue
			}

			// check if we are done with scanning the history of all streams
			if scanHistory {
				scanHistory = false

				// Getting history always returns all streams in the same order as queried
				// (even a stream that doesn't have any history left, in that case redis returns an empty slice)
				// Thus, we can use a simple incrementing index to get the streamArg for a stream in the response
				x := 0
				for _, stream := range resReadStream {
					// If the stream had messages in the history, continue scanning after the latest read message.
					if len(stream.Messages) > 0 {
						scanHistory = true
						streamsArg[streamLen+x] = stream.Messages[len(stream.Messages)-1].ID

						c.pushInfo(fmt.Sprintf(
							"stream %q had %d more messages in the history (delivered but no yet acked),"+
								"continuing scanning after %q",
							stream.Stream,
							len(stream.Messages),
							streamsArg[streamLen+x],
						))
					}
					x++
				}

				if !scanHistory {
					c.pushInfo("completed scan of history")

					// Update stream args to read latest messages for all streams
					for j := 0; j < streamLen; j++ {
						streamsArg[streamLen+j] = ">"
					}

					continue
				}
			}

			// reset fail count
			consecutiveFailures = 0

			// if no messages were read we can skip iteration
			if len(resReadStream) == 0 {
				continue
			}

			// retrieve all messages across all streams and put them into the message queue
			for _, stream := range resReadStream {
				for _, m := range stream.Messages {
					c.messageQueue <- message{
						streamID: stream.Stream,
						id:       m.ID,
						values:   m.Values,
					}
				}
			}
		}
	}
}

// reclaimer periodically inspects pending messages with XPENDING command.
// If a message sits longer than processingTimeout, we attempt to reclaim the message for this consumer
// and enqueue it for processing.
//
//nolint:funlen,gocognit // refactor if needed
func (c *RedisConsumer) reclaimer(ctx context.Context, reclaimInterval time.Duration) {
	reclaimTimer := time.NewTimer(reclaimInterval)
	defer func() {
		reclaimTimer.Stop()
	}()

	const (
		baseCount = 16
		maxCount  = 1024
	)

	// the minimum message ID which we are querying for.
	// redis treats "-" as smaller than any valid message ID
	start := "-"
	// the maximum message ID which we are querying for.
	// redis treats "+" as bigger than any valid message ID
	end := "+"
	count := baseCount

	for {
		select {
		case <-ctx.Done():
			return
		case <-reclaimTimer.C:
			for streamID, handler := range c.streams {
				resPending, errPending := c.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
					Stream: streamID,
					Group:  c.groupName,
					Start:  start,
					End:    end,
					Idle:   handler.config.idleTimeout,
					Count:  int64(count),
				}).Result()
				if errPending != nil && !errors.Is(errPending, redis.Nil) {
					c.pushError(fmt.Errorf("failed to fetch pending messages: %w", errPending))
					continue
				}

				if len(resPending) == 0 {
					continue
				}

				// It's safe to change start of the requested range for the next iteration to oldest message.
				start = resPending[0].ID

				for _, resMessage := range resPending {
					if resMessage.RetryCount > int64(handler.config.maxRetries) {
						// Retry count gets increased after every XCLAIM.
						// Large retry count might mean there is something wrong with the message, so we'll XACK it.
						// WARNING this will discard the message!
						errAck := c.rdb.XAck(ctx, streamID, c.groupName, resMessage.ID).Err()
						if errAck != nil {
							c.pushError(fmt.Errorf(
								"failed to force acknowledge (discard) message '%s' (Retries: %d) in stream '%s': %w",
								resMessage.ID, resMessage.RetryCount, streamID, errAck))
						} else {
							retryCount := resMessage.RetryCount - 1 // redis is counting this execution as retry
							c.pushError(fmt.Errorf(
								"force acknowledged (discarded) message '%s' (Retries: %d) in stream '%s'",
								resMessage.ID, retryCount, streamID))
						}
						continue
					}

					// Otherwise, claim the message so we can retry it.
					claimedMessages, errClaim := c.rdb.XClaim(ctx, &redis.XClaimArgs{
						Stream:   streamID,
						Group:    c.groupName,
						Consumer: c.consumerName,
						MinIdle:  handler.config.idleTimeout,
						Messages: []string{resMessage.ID},
					}).Result()

					if errors.Is(errClaim, redis.Nil) {
						// Receiving redis.Nil here means the message is removed from the stream (because of MAXLEN).
						// The only option is to acknowledge it with XACK.
						errAck := c.rdb.XAck(ctx, streamID, c.groupName, resMessage.ID).Err()
						if errAck != nil {
							c.pushError(fmt.Errorf("failed to acknowledge failed message '%s' in stream '%s': %w",
								resMessage.ID, streamID, errAck))
						} else {
							c.pushInfo(fmt.Sprintf("acknowledged failed message '%s' in stream '%s'",
								resMessage.ID, streamID))
						}

						continue
					}

					if errClaim != nil {
						// This can happen if two consumers try to claim the same message at once.
						// One would succeed and the other will get an error.
						c.pushError(fmt.Errorf("failed to claim message '%s' in stream '%s': %w",
							resMessage.ID, streamID, errClaim))

						continue
					}

					// This is not expected to happen (message will be retried or eventually discarded)
					if len(claimedMessages) == 0 {
						c.pushError(fmt.Errorf(
							"no error when claiming message '%s' in stream '%s', but redis returned no message",
							resMessage.ID, streamID))

						continue
					}

					// we claimed only one message id so there is only one message in the slice
					claimedMessage := claimedMessages[0]
					c.messageQueue <- message{
						streamID: streamID,
						id:       claimedMessage.ID,
						values:   claimedMessage.Values,
					}
				}

				// If number of messages that we got is equal to the number that we requested
				// it means that there's a lot for processing, so we'll increase number of messages
				// that we'll pull in the next iteration.
				if len(resPending) == count {
					count *= 2
					if count > maxCount {
						count = maxCount
					}
				} else {
					count = baseCount
				}
			}

			reclaimTimer.Reset(reclaimInterval)
		}
	}
}

// consumer method consumes messages coming from Redis. The method terminates when messageQueue channel closes.
func (c *RedisConsumer) consumer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-c.messageQueue:
			if m.id == "" {
				// id should never be empty, if it is then the channel is closed
				return
			}

			handler, ok := c.streams[m.streamID]
			if !ok {
				// we don't want to ack the message
				// maybe someone else can claim and process it (worst case it expires)
				c.pushError(fmt.Errorf("received message '%s' in stream '%s' that doesn't belong to us, skip",
					m.id, m.streamID))
				continue
			}

			err := func() (err error) {
				// Ensure that handlers don't cause panic.
				defer func() {
					if r := recover(); r != nil {
						c.pushError(fmt.Errorf("PANIC when processing message '%s' in stream '%s':\n%s",
							m.id, m.streamID, debug.Stack()))
					}
				}()

				return handler.handle(ctx, m.id, m.values)
			}()
			if err != nil {
				c.pushError(fmt.Errorf("failed to process message '%s' in stream '%s': %w", m.id, m.streamID, err))
				continue
			}

			err = c.rdb.XAck(ctx, m.streamID, c.groupName, m.id).Err()
			if err != nil {
				c.pushError(fmt.Errorf("failed to acknowledge message '%s' in stream '%s': %w", m.id, m.streamID, err))
				continue
			}
		}
	}
}

func (c *RedisConsumer) removeStaleConsumers(ctx context.Context, maxAge time.Duration) {
	for streamID := range c.streams {
		// Fetch all consumers for this stream and group.
		resConsumers, err := c.rdb.XInfoConsumers(ctx, streamID, c.groupName).Result()
		if err != nil {
			c.pushError(fmt.Errorf("failed to read consumers for stream '%s': %w", streamID, err))
			return
		}

		// Delete old consumers, but only if they don't have pending messages.
		for _, resConsumer := range resConsumers {
			age := time.Duration(resConsumer.Idle) * time.Millisecond
			if resConsumer.Pending > 0 || age < maxAge {
				continue
			}

			err = c.rdb.XGroupDelConsumer(ctx, streamID, c.groupName, resConsumer.Name).Err()
			if err != nil {
				c.pushError(fmt.Errorf(
					"failed to remove stale consumer '%s' from group '%s' (age '%s') for stream '%s': %w",
					resConsumer.Name, c.groupName, age, streamID, err))
				continue
			}

			c.pushInfo(fmt.Sprintf("removed stale consumer '%s' from group '%s' (age '%s') for stream '%s'",
				resConsumer.Name, c.groupName, age, streamID))
		}
	}
}

func (c *RedisConsumer) pushError(err error) {
	select {
	case c.errorCh <- err:
	default:
	}
}

func (c *RedisConsumer) pushInfo(s string) {
	select {
	case c.infoCh <- s:
	default:
	}
}

func (c *RedisConsumer) Errors() <-chan error { return c.errorCh }
func (c *RedisConsumer) Infos() <-chan string { return c.infoCh }

func (c *RedisConsumer) createGroupForAllStreams(ctx context.Context) error {
	for streamID := range c.streams {
		err := createGroup(ctx, c.rdb, streamID, c.groupName)
		if err != nil {
			return err
		}
	}

	return nil
}

func createGroup(ctx context.Context, rdb redis.UniversalClient, streamID string, groupName string) error {
	// Creates a new consumer group that starts receiving messages from now on.
	// Existing messges in the queue are ignored (we don't want to overload a group with old messages)
	// For more details of the XGROUPCREATE api see https://redis.io/commands/xgroup-create/
	err := rdb.XGroupCreateMkStream(ctx, streamID, groupName, "$").Err()
	if err != nil && !strings.HasPrefix(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group '%s' for stream '%s': %w", groupName, streamID, err)
	}

	return nil
}
