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

package events

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
)

// ReaderFactoryFunc is an abstraction of a factory method that creates customized Reader implementations (type [R]).
// It is triggered by the ReaderFactory to create a new instance of the Reader to launch.
// The provided GenericReader object is available exclusively to the factory method (every call has a fresh instance)
// and should be used as base of any custom Reader implementation (use ReaderRegisterEvent to register custom handler).
type ReaderFactoryFunc[R Reader] func(reader *GenericReader) (R, error)

// ReaderFactory allows to launch event readers of type [R] (can be GenericReader or customized readers).
type ReaderFactory[R Reader] struct {
	category                string
	streamConsumerFactoryFn StreamConsumerFactoryFunc
	readerFactoryFn         ReaderFactoryFunc[R]
}

// Launch launches a new reader for the provided group and client name.
// The setup method should be used to register the different events the reader will act on.
// To stop the reader and cleanup its resources the returned ReaderCanceler can be used.
// The reader also cancels automatically when the provided context is canceled.
// NOTE: Do not setup the reader outside of the setup method!
func (f *ReaderFactory[R]) Launch(ctx context.Context,
	groupName string, readerName string, setup func(R) error) (*ReaderCanceler, error) {
	if groupName == "" {
		return nil, errors.New("groupName can't be empty")
	}
	if setup == nil {
		return nil, errors.New("setup function can't be nil")
	}

	// setup ctx with copied logger that has extra fields set
	log := log.Ctx(ctx).With().
		Str("events.category", f.category).
		Str("events.group_name", groupName).
		Str("events.reader_name", readerName).
		Logger()

	// create new stream consumer using factory method
	streamConsumer, err := f.streamConsumerFactoryFn(groupName, readerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create new stream consumer: %w", err)
	}

	// create generic reader object
	innerReader := &GenericReader{
		streamConsumer: streamConsumer,
		category:       f.category,
	}

	// create new reader (could return the innerReader itself, but also allows to launch customized readers)
	reader, err := f.readerFactoryFn(innerReader)
	if err != nil {
		//nolint:gocritic // only way to achieve this AFAIK - lint proposal is not building
		return nil, fmt.Errorf("failed creation of event reader of type %T: %w", *new(R), err)
	}

	// execute setup function on reader (will configure concurrency, processingTimeout, ..., and register handlers)
	err = setup(reader)
	if err != nil {
		return nil, fmt.Errorf("failed custom setup of event reader: %w", err)
	}

	// hook into all available logs
	go func(errorCh <-chan error) {
		for err := range errorCh {
			log.Err(err).Msg("received an error from stream consumer")
		}
	}(streamConsumer.Errors())

	go func(infoCh <-chan string) {
		for s := range infoCh {
			log.Info().Msgf("stream consumer: %s", s)
		}
	}(streamConsumer.Infos())

	// prepare context (inject logger and make canceable)
	ctx = log.WithContext(ctx)
	ctx, cancelFn := context.WithCancel(ctx)

	// start consumer
	err = innerReader.streamConsumer.Start(ctx)
	if err != nil {
		cancelFn()
		return nil, fmt.Errorf("failed to start consumer: %w", err)
	}

	return &ReaderCanceler{
		cancelFn: func() error {
			cancelFn()
			return nil
		},
	}, nil
}

// ReaderCanceler exposes the functionality to cancel a reader explicitly.
type ReaderCanceler struct {
	canceled bool
	cancelFn func() error
}

func (d *ReaderCanceler) Cancel() error {
	if d.canceled {
		return errors.New("reader has already been canceled")
	}

	// call cancel (might be async)
	err := d.cancelFn()
	if err != nil {
		return fmt.Errorf("failed to cancel reader: %w", err)
	}

	d.canceled = true

	return nil
}

// Reader specifies the minimum functionality a reader should expose.
// NOTE: we don't want to enforce any event registration methods here, allowing full control for customized readers.
type Reader interface {
	Configure(opts ...ReaderOption)
}

type HandlerFunc[T interface{}] func(context.Context, *Event[T]) error

// GenericReader represents an event reader that supports registering type safe handlers
// for an arbitrary set of custom events within a given event category using the ReaderRegisterEvent method.
// NOTE: Optimally this should be an interface with RegisterEvent[T] method, but that's currently not possible in go.
// IMPORTANT: This reader should not be instantiated from external packages.
type GenericReader struct {
	streamConsumer StreamConsumer
	category       string
}

// ReaderRegisterEvent registers a type safe handler function on the reader for a specific event.
// This method allows to register type safe handlers without the need of handling the raw stream payload.
// NOTE: Generic arguments are not allowed for struct methods, hence pass the reader as input parameter.
func ReaderRegisterEvent[T interface{}](reader *GenericReader,
	eventType EventType, fn HandlerFunc[T], opts ...HandlerOption) error {
	streamID := getStreamID(reader.category, eventType)

	// register handler for event specific stream.
	return reader.streamConsumer.Register(streamID,
		func(ctx context.Context, messageID string, streamPayload map[string]interface{}) error {
			if streamPayload == nil {
				return fmt.Errorf("stream payload is nil for message '%s'", messageID)
			}

			// retrieve event from stream payload
			eventRaw, ok := streamPayload[streamPayloadKey]
			if !ok {
				return fmt.Errorf("stream payload doesn't contain event (key: '%s') for message '%s'", streamPayloadKey, messageID)
			}

			// retrieve bytes from raw event
			// NOTE: Redis returns []byte as string - to avoid unnecessary conversion we handle both types here.
			var eventBytes []byte
			switch v := eventRaw.(type) {
			case string:
				eventBytes = []byte(v)
			case []byte:
				eventBytes = v
			default:
				return fmt.Errorf("stream payload is not of expected type string or []byte but of type %T (message '%s')",
					eventRaw, messageID)
			}

			// decode event to correct type
			var event Event[T]
			decoder := gob.NewDecoder(bytes.NewReader(eventBytes))
			err := decoder.Decode(&event)
			if err != nil {
				//nolint:gocritic // only way to achieve this AFAIK - lint proposal is not building
				return fmt.Errorf("stream payload can't be decoded into type %T (message '%s')", *new(T), messageID)
			}

			// populate event ID using the message ID (has to be populated here, producer doesn't know the message ID yet)
			event.ID = messageID

			// update ctx with event type for proper logging
			log := log.Ctx(ctx).With().
				Str("events.type", string(eventType)).
				Str("events.id", event.ID).
				Logger()
			ctx = log.WithContext(ctx)

			// call provided handler with correctly typed payload
			err = fn(ctx, &event)

			// handle discardEventError
			if errors.Is(err, errDiscardEvent) {
				log.Warn().Err(err).Msgf("discarding event '%s'", event.ID)
				return nil
			}

			// any other error we return as is
			return err
		}, toStreamHandlerOptions(opts)...)
}

func (r *GenericReader) Configure(opts ...ReaderOption) {
	r.streamConsumer.Configure(toStreamConsumerOptions(opts)...)
}
