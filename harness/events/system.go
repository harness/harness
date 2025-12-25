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
