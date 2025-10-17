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
	"runtime/debug"
	"sync"
	"time"
)

// memoryMessage extends the message object to allow tracking retries.
type memoryMessage struct {
	message
	retries int64
}

// MemoryConsumer consumes streams from a MemoryBroker.
type MemoryConsumer struct {
	broker *MemoryBroker
	// namespace specifies the namespace of the keys - any stream key will be prefixed with it
	namespace string
	// groupName specifies the name of the consumer group.
	groupName string

	// Config is the generic consumer configuration.
	Config ConsumerConfig

	// streams is a map of all registered streams and their handlers.
	streams map[string]handler

	isStarted    bool
	messageQueue chan memoryMessage
	errorCh      chan error
	infoCh       chan string
}

func NewMemoryConsumer(broker *MemoryBroker, namespace string, groupName string) (*MemoryConsumer, error) {
	if groupName == "" {
		return nil, errors.New("groupName can't be empty")
	}

	const queueCapacity = 500
	const errorChCapacity = 64
	const infoChCapacity = 64

	return &MemoryConsumer{
		broker:       broker,
		namespace:    namespace,
		groupName:    groupName,
		streams:      map[string]handler{},
		Config:       defaultConfig,
		isStarted:    false,
		messageQueue: make(chan memoryMessage, queueCapacity),
		errorCh:      make(chan error, errorChCapacity),
		infoCh:       make(chan string, infoChCapacity),
	}, nil
}

func (c *MemoryConsumer) Configure(opts ...ConsumerOption) {
	if c.isStarted {
		return
	}

	for _, opt := range opts {
		opt.apply(&c.Config)
	}
}

func (c *MemoryConsumer) Register(streamID string, fn HandlerFunc, opts ...HandlerOption) error {
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
		return fmt.Errorf("consumer is already registered for '%s' (full stream '%s')", streamID, transposedStreamID)
	}

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

func (c *MemoryConsumer) Start(ctx context.Context) error {
	if c.isStarted {
		return ErrAlreadyStarted
	}

	if len(c.streams) == 0 {
		return errors.New("no streams registered")
	}

	// mark as started before starting go routines (can't error out from here)
	c.isStarted = true

	wg := &sync.WaitGroup{}

	// start routines to read messages from broker
	for streamID := range c.streams {
		wg.Add(1)
		go func(stream string) {
			defer wg.Done()
			c.reader(ctx, stream)
		}(streamID)
	}

	// start workers
	for i := 0; i < c.Config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.consume(ctx)
		}()
	}

	// start cleanup routing
	go func() {
		// wait for all go routines to complete
		wg.Wait()

		close(c.messageQueue)
		close(c.infoCh)
		close(c.errorCh)
	}()

	return nil
}

// reader reads the messages of a specific stream from the broker and puts it
// into the single message queue monitored by the consumers.
func (c *MemoryConsumer) reader(ctx context.Context, streamID string) {
	streamQueue := c.broker.messages(streamID, c.groupName)
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-streamQueue:
			c.messageQueue <- memoryMessage{
				message: m,
				retries: 0,
			}
		}
	}
}

func (c *MemoryConsumer) consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-c.messageQueue:
			handler, ok := c.streams[m.streamID]
			if !ok {
				// we only take messages from registered streams, this should never happen.
				// WARNING this will discard the message
				c.pushError(fmt.Errorf("discard message with id '%s' from stream '%s' - doesn't belong to us",
					m.id, m.streamID))
				continue
			}

			c.processMessage(ctx, handler, m)
		}
	}
}

func (c *MemoryConsumer) processMessage(ctx context.Context, handler handler, m memoryMessage) {
	var handlingErr error

	ctxWithCancel, cancelFn := context.WithCancel(ctx)
	defer func(err error) {
		// If the original execution errors out, we rely on the timeout to retry. This is to keep the behaviour same
		// as the redis consumer.
		if err == nil {
			cancelFn()
		}
	}(handlingErr)

	// Start a retry goroutine with `idleTimeout` delay
	go c.retryPostTimeout(ctxWithCancel, handler, m)

	handlingErr = func() (err error) {
		// Ensure that handlers don't cause panic.
		defer func() {
			if r := recover(); r != nil {
				c.pushError(fmt.Errorf("PANIC when processing message '%s' in stream '%s':\n%s",
					m.id, m.streamID, debug.Stack()))
			}
		}()

		return handler.handle(ctx, m.id, m.values)
	}()

	if handlingErr != nil {
		c.pushError(fmt.Errorf("failed to process message with id '%s' in stream '%s' (retries: %d): %w",
			m.id, m.streamID, m.retries, handlingErr))
	}
}

func (c *MemoryConsumer) retryPostTimeout(ctxWithCancel context.Context, handler handler, m memoryMessage) {
	timer := time.NewTimer(handler.config.idleTimeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		c.retryMessage(m, handler.config.maxRetries)
	case <-ctxWithCancel.Done():
		// Retry canceled if message is processed
		// Drain the timer channel if it is already stopped
		if !timer.Stop() {
			<-timer.C
		}
	}
}

func (c *MemoryConsumer) retryMessage(m memoryMessage, maxRetries int) {
	if m.retries >= int64(maxRetries) {
		c.pushError(fmt.Errorf("discard message with id '%s' from stream '%s' - failed %d retries",
			m.id, m.streamID, m.retries))
		return
	}

	// increase retry count
	m.retries++

	// requeue message for a retry (needs to be in a separate go func to avoid deadlock)
	// IMPORTANT: this won't requeue to broker, only in this consumer's queue!
	go func() {
		// TODO: linear/exponential backoff relative to retry count might be good
		c.messageQueue <- m
	}()
}

func (c *MemoryConsumer) Errors() <-chan error { return c.errorCh }
func (c *MemoryConsumer) Infos() <-chan string { return c.infoCh }

func (c *MemoryConsumer) pushError(err error) {
	select {
	case c.errorCh <- err:
	default:
	}
}
