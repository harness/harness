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
	"context"

	"github.com/harness/gitness/stream"
)

// StreamProducer is an abstraction of a producer from the streams package.
type StreamProducer interface {
	Send(ctx context.Context, streamID string, payload map[string]any) (string, error)
}

// StreamConsumer is an abstraction of a consumer from the streams package.
type StreamConsumer interface {
	Register(streamID string, handler stream.HandlerFunc, opts ...stream.HandlerOption) error
	Configure(opts ...stream.ConsumerOption)
	Start(ctx context.Context) error
	Errors() <-chan error
	Infos() <-chan string
}

// StreamConsumerFactoryFunc is an abstraction of a factory method for stream consumers.
type StreamConsumerFactoryFunc func(groupName string, consumerName string) (StreamConsumer, error)
