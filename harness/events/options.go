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
	"time"

	"github.com/harness/gitness/stream"
)

/*
 * Expose event package options to simplify usage for consumers by hiding the stream package.
 * Since we only forward the options, event options are simply aliases of stream options.
 */

// ReaderOption can be used to configure event readers.
type ReaderOption stream.ConsumerOption

func toStreamConsumerOptions(opts []ReaderOption) []stream.ConsumerOption {
	streamOpts := make([]stream.ConsumerOption, len(opts))
	for i, opt := range opts {
		streamOpts[i] = stream.ConsumerOption(opt)
	}
	return streamOpts
}

// WithConcurrency sets up the concurrency of the reader.
func WithConcurrency(concurrency int) ReaderOption {
	return stream.WithConcurrency(concurrency)
}

// WithHandlerOptions sets up the default options for event handlers.
func WithHandlerOptions(opts ...HandlerOption) ReaderOption {
	return stream.WithHandlerOptions(toStreamHandlerOptions(opts)...)
}

// HandlerOption can be used to configure event handlers.
type HandlerOption stream.HandlerOption

func toStreamHandlerOptions(opts []HandlerOption) []stream.HandlerOption {
	streamOpts := make([]stream.HandlerOption, len(opts))
	for i, opt := range opts {
		streamOpts[i] = stream.HandlerOption(opt)
	}
	return streamOpts
}

// WithMaxRetries can be used to set the max retry count for a specific event handler.
func WithMaxRetries(maxRetries int) HandlerOption {
	return stream.WithMaxRetries(maxRetries)
}

// WithIdleTimeout can be used to set the idle timeout for a specific event handler.
func WithIdleTimeout(timeout time.Duration) HandlerOption {
	return stream.WithIdleTimeout(timeout)
}
