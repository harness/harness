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
	"testing"
	"time"
)

func TestWithConcurrency(t *testing.T) {
	t.Run("valid concurrency", func(t *testing.T) {
		config := &ConsumerConfig{}
		opt := WithConcurrency(10)
		opt.apply(config)

		if config.Concurrency != 10 {
			t.Errorf("expected concurrency 10, got %d", config.Concurrency)
		}
	})

	t.Run("minimum valid concurrency", func(t *testing.T) {
		config := &ConsumerConfig{}
		opt := WithConcurrency(1)
		opt.apply(config)

		if config.Concurrency != 1 {
			t.Errorf("expected concurrency 1, got %d", config.Concurrency)
		}
	})

	t.Run("maximum valid concurrency", func(t *testing.T) {
		config := &ConsumerConfig{}
		opt := WithConcurrency(MaxConcurrency)
		opt.apply(config)

		if config.Concurrency != MaxConcurrency {
			t.Errorf("expected concurrency %d, got %d", MaxConcurrency, config.Concurrency)
		}
	})

	t.Run("zero concurrency panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for zero concurrency")
			}
		}()
		WithConcurrency(0)
	})

	t.Run("negative concurrency panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for negative concurrency")
			}
		}()
		WithConcurrency(-1)
	})

	t.Run("concurrency above max panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for concurrency above max")
			}
		}()
		WithConcurrency(MaxConcurrency + 1)
	})
}

func TestWithMaxRetries(t *testing.T) {
	t.Run("valid max retries", func(t *testing.T) {
		config := &HandlerConfig{}
		opt := WithMaxRetries(5)
		opt.apply(config)

		if config.maxRetries != 5 {
			t.Errorf("expected maxRetries 5, got %d", config.maxRetries)
		}
	})

	t.Run("zero max retries", func(t *testing.T) {
		config := &HandlerConfig{}
		opt := WithMaxRetries(0)
		opt.apply(config)

		if config.maxRetries != 0 {
			t.Errorf("expected maxRetries 0, got %d", config.maxRetries)
		}
	})

	t.Run("maximum valid retries", func(t *testing.T) {
		config := &HandlerConfig{}
		opt := WithMaxRetries(MaxMaxRetries)
		opt.apply(config)

		if config.maxRetries != MaxMaxRetries {
			t.Errorf("expected maxRetries %d, got %d", MaxMaxRetries, config.maxRetries)
		}
	})

	t.Run("negative max retries panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for negative max retries")
			}
		}()
		WithMaxRetries(-1)
	})

	t.Run("max retries above max panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for max retries above max")
			}
		}()
		WithMaxRetries(MaxMaxRetries + 1)
	})
}

func TestWithIdleTimeout(t *testing.T) {
	t.Run("valid idle timeout", func(t *testing.T) {
		config := &HandlerConfig{}
		timeout := 10 * time.Second
		opt := WithIdleTimeout(timeout)
		opt.apply(config)

		if config.idleTimeout != timeout {
			t.Errorf("expected idleTimeout %v, got %v", timeout, config.idleTimeout)
		}
	})

	t.Run("minimum valid idle timeout", func(t *testing.T) {
		config := &HandlerConfig{}
		opt := WithIdleTimeout(MinIdleTimeout)
		opt.apply(config)

		if config.idleTimeout != MinIdleTimeout {
			t.Errorf("expected idleTimeout %v, got %v", MinIdleTimeout, config.idleTimeout)
		}
	})

	t.Run("idle timeout below minimum panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for idle timeout below minimum")
			}
		}()
		WithIdleTimeout(MinIdleTimeout - 1*time.Second)
	})

	t.Run("zero idle timeout panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for zero idle timeout")
			}
		}()
		WithIdleTimeout(0)
	})
}

func TestWithHandlerOptions(t *testing.T) {
	t.Run("applies single handler option", func(t *testing.T) {
		config := &ConsumerConfig{}
		opt := WithHandlerOptions(WithMaxRetries(10))
		opt.apply(config)

		if config.DefaultHandlerConfig.maxRetries != 10 {
			t.Errorf("expected maxRetries 10, got %d", config.DefaultHandlerConfig.maxRetries)
		}
	})

	t.Run("applies multiple handler options", func(t *testing.T) {
		config := &ConsumerConfig{}
		timeout := 10 * time.Second
		opt := WithHandlerOptions(
			WithMaxRetries(5),
			WithIdleTimeout(timeout),
		)
		opt.apply(config)

		if config.DefaultHandlerConfig.maxRetries != 5 {
			t.Errorf("expected maxRetries 5, got %d", config.DefaultHandlerConfig.maxRetries)
		}
		if config.DefaultHandlerConfig.idleTimeout != timeout {
			t.Errorf("expected idleTimeout %v, got %v", timeout, config.DefaultHandlerConfig.idleTimeout)
		}
	})

	t.Run("applies no handler options", func(t *testing.T) {
		config := &ConsumerConfig{}
		opt := WithHandlerOptions()
		opt.apply(config)

		// Should not panic and should leave config unchanged
		if config.DefaultHandlerConfig.maxRetries != 0 {
			t.Errorf("expected maxRetries 0, got %d", config.DefaultHandlerConfig.maxRetries)
		}
	})
}
