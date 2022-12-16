// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import "errors"

// System represents a single contained event system that is used
// to setup event Reporters and ReaderFactories.
type System struct {
	streamConsumerFactoryFn StreamConsumerFactoryFunc
	streamProducer          StreamProducer
}

func NewSystem(streamConsumerFactoryFunc StreamConsumerFactoryFunc, streamProducer StreamProducer) (*System, error) {
	if streamConsumerFactoryFunc == nil {
		return nil, errors.New("streamConsumerFactoryFunc can't be empty")
	}
	if streamProducer == nil {
		return nil, errors.New("streamProducer can't be empty")
	}

	return &System{
		streamConsumerFactoryFn: streamConsumerFactoryFunc,
		streamProducer:          streamProducer,
	}, nil
}

func NewReaderFactory[R Reader](system *System, category string, fn ReaderFactoryFunc[R]) (*ReaderFactory[R], error) {
	if system == nil {
		return nil, errors.New("system can't be empty")
	}
	if category == "" {
		return nil, errors.New("category can't be empty")
	}
	if fn == nil {
		return nil, errors.New("fn can't be empty")
	}

	return &ReaderFactory[R]{
		// values coming from system
		streamConsumerFactoryFn: system.streamConsumerFactoryFn,

		// values coming from input parameters
		category:        category,
		readerFactoryFn: fn,
	}, nil
}

func NewReporter(system *System, category string) (*GenericReporter, error) {
	if system == nil {
		return nil, errors.New("system can't be empty")
	}
	if category == "" {
		return nil, errors.New("category can't be empty")
	}

	return &GenericReporter{
		// values coming from system
		producer: system.streamProducer,

		// values coming from input parameters
		category: category,
	}, nil
}
