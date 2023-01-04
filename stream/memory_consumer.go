// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
	namespace     string
	concurrency   int
	maxRetryCount int64
	groupName     string
	streams       map[string]HandlerFunc

	state        consumerState
	messageQueue chan memoryMessage
	errorCh      chan error
	infoCh       chan string
}

func NewMemoryConsumer(broker *MemoryBroker, namespace string, groupName string) *MemoryConsumer {
	const queueCapacity = 500
	const errorChCapacity = 64
	const infoChCapacity = 64
	const concurrency = 1
	return &MemoryConsumer{
		broker:       broker,
		namespace:    namespace,
		concurrency:  concurrency,
		groupName:    groupName,
		streams:      make(map[string]HandlerFunc),
		state:        consumerStateSetup,
		messageQueue: make(chan memoryMessage, queueCapacity),
		errorCh:      make(chan error, errorChCapacity),
		infoCh:       make(chan string, infoChCapacity),
	}
}

func (c *MemoryConsumer) Register(streamID string, handler HandlerFunc) error {
	if err := checkConsumerStateTransition(c.state, consumerStateSetup); err != nil {
		return err
	}

	if streamID == "" {
		return errors.New("streamID can't be empty")
	}
	if handler == nil {
		return errors.New("handler can't be empty")
	}

	// transpose streamID to key namespace - no need to keep inner streamID
	transposedStreamID := transposeStreamID(c.namespace, streamID)
	if _, ok := c.streams[transposedStreamID]; ok {
		return fmt.Errorf("consumer is already registered for '%s' (full stream '%s')", streamID, transposedStreamID)
	}

	c.streams[transposedStreamID] = handler
	return nil
}

func (c *MemoryConsumer) SetConcurrency(concurrency int) error {
	if err := checkConsumerStateTransition(c.state, consumerStateSetup); err != nil {
		return err
	}

	if concurrency < 1 || concurrency > MaxConcurrency {
		return fmt.Errorf("concurrency has to be between 1 and %d (inclusive)", MaxConcurrency)
	}

	c.concurrency = concurrency

	return nil
}

func (c *MemoryConsumer) SetMaxRetryCount(retryCount int64) error {
	if err := checkConsumerStateTransition(c.state, consumerStateSetup); err != nil {
		return err
	}

	if retryCount < 1 || retryCount > MaxRetryCount {
		return fmt.Errorf("max retry count has to be between 1 and %d (inclusive)", MaxRetryCount)
	}

	c.maxRetryCount = retryCount

	return nil
}

func (c *MemoryConsumer) SetProcessingTimeout(timeout time.Duration) error {
	if err := checkConsumerStateTransition(c.state, consumerStateSetup); err != nil {
		return err
	}

	// we don't have an idle timeout for this implementation

	return nil
}

func (c *MemoryConsumer) Start(ctx context.Context) error {
	if err := checkConsumerStateTransition(c.state, consumerStateStarted); err != nil {
		return err
	}

	if len(c.streams) == 0 {
		return errors.New("no streams registered")
	}

	// update state to started before starting go routines (can't error out from here)
	c.state = consumerStateStarted

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
	for i := 0; i < c.concurrency; i++ {
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

		// update state to finished
		c.state = consumerStateFinished

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
			fn, ok := c.streams[m.streamID]
			if !ok {
				// we only take messages from registered streams, this should never happen.
				// WARNING this will discard the message
				c.pushError(fmt.Errorf("discard message with id '%s' from stream '%s' - doesn't belong to us",
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

				return fn(ctx, m.id, m.values)
			}()

			if err != nil {
				c.pushError(fmt.Errorf("failed to process message with id '%s' in stream '%s' (retries: %d): %w",
					m.id, m.streamID, m.retries, err))

				if m.retries >= c.maxRetryCount {
					c.pushError(fmt.Errorf(
						"discard message with id '%s' from stream '%s' - failed %d retries",
						m.id, m.streamID, m.retries))
					continue
				}

				// increase retry count
				m.retries++

				// requeue message for a retry (needs to be in a separate go func to avoid deadlock)
				// IMPORTANT: this won't requeue to broker, only in this consumer's queue!
				go func() {
					// TODO: linear/exponential backoff relative to retry count might be good
					time.Sleep(5 * time.Second)
					c.messageQueue <- m
				}()
			}
		}
	}
}

func (c *MemoryConsumer) Errors() <-chan error { return c.errorCh }
func (c *MemoryConsumer) Infos() <-chan string { return c.infoCh }

func (c *MemoryConsumer) pushError(err error) {
	select {
	case c.errorCh <- err:
	default:
	}
}
