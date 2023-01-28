// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
