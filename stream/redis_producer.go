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
	"io"
	"maps"
	"net"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// Defaults sized to cover GCP Memorystore STANDARD_HA switchovers (~6s observed)
// with margin, matching harness-core Events Framework RedisProducer (~6 attempts / ~13s).
const (
	DefaultProducerMaxAttempts    = 6
	DefaultProducerInitialBackoff = 1 * time.Second
	DefaultProducerBackoffFactor  = 1.5
	DefaultProducerMaxBackoff     = 8 * time.Second
)

// ProducerRetryConfig controls RedisProducer Send retries for transient failures.
// Zero values are replaced with defaults via Normalize.
type ProducerRetryConfig struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	BackoffFactor  float64
	MaxBackoff     time.Duration
}

// DefaultProducerRetryConfig returns the Memorystore-failover-sized defaults.
func DefaultProducerRetryConfig() ProducerRetryConfig {
	return ProducerRetryConfig{
		MaxAttempts:    DefaultProducerMaxAttempts,
		InitialBackoff: DefaultProducerInitialBackoff,
		BackoffFactor:  DefaultProducerBackoffFactor,
		MaxBackoff:     DefaultProducerMaxBackoff,
	}
}

// Normalize fills zero/invalid fields with defaults.
func (c ProducerRetryConfig) Normalize() ProducerRetryConfig {
	d := DefaultProducerRetryConfig()
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = d.MaxAttempts
	}
	if c.InitialBackoff <= 0 {
		c.InitialBackoff = d.InitialBackoff
	}
	if c.BackoffFactor <= 1 {
		c.BackoffFactor = d.BackoffFactor
	}
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = d.MaxBackoff
	}
	return c
}

// ProducerOption configures optional RedisProducer behavior.
type ProducerOption func(*RedisProducer)

// WithProducerRetry sets Send retry parameters (zeros normalized to defaults).
func WithProducerRetry(cfg ProducerRetryConfig) ProducerOption {
	return func(p *RedisProducer) {
		p.retry = cfg.Normalize()
	}
}

type RedisProducer struct {
	rdb redis.UniversalClient
	// namespace defines the namespace of the stream keys - any stream key will be prefixed with it.
	namespace string
	// maxStreamLength defines the maximum number of entries in each stream (ring buffer).
	maxStreamLength int64
	// approxMaxStreamLength specifies whether the maxStreamLength should be approximated.
	// NOTE: enabling approximation of stream length can lead to performance improvements.
	approxMaxStreamLength bool
	retry                 ProducerRetryConfig
}

func NewRedisProducer(rdb redis.UniversalClient, namespace string,
	maxStreamLength int64, approxMaxStreamLength bool, opts ...ProducerOption) *RedisProducer {
	p := &RedisProducer{
		rdb:                   rdb,
		namespace:             namespace,
		maxStreamLength:       maxStreamLength,
		approxMaxStreamLength: approxMaxStreamLength,
		retry:                 DefaultProducerRetryConfig(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Send starts a background goroutine that writes to the Redis stream with retries
// and returns immediately. The caller context's values are preserved via
// context.WithoutCancel so request cancel/deadline does not abort failover backoff.
// The returned message ID is always empty; callers that need a stable ID should use
// one from the payload (e.g. events.Event.ID). Retried XADD with ID "*" is at-least-once.
// TODO(CODE-6402): replace fire-and-forget goroutine with a transactional outbox
// (persist then dispatch) for durable produce and bounded concurrency.
func (p *RedisProducer) Send(ctx context.Context, streamID string, payload map[string]any) (string, error) {
	bgCtx := context.WithoutCancel(ctx)
	streamIDCopy := streamID
	payloadCopy := maps.Clone(payload)
	go func() {
		if err := p.sendWithRetry(bgCtx, streamIDCopy, payloadCopy); err != nil {
			log.Error().Err(err).
				Str("stream", streamIDCopy).
				Msg("background redis produce failed after retries")
		}
	}()
	return "", nil
}

// sendWithRetry performs XADD with exponential backoff.
func (p *RedisProducer) sendWithRetry(ctx context.Context, streamID string, payload map[string]any) error {
	transposedStreamID := transposeStreamID(p.namespace, streamID)

	args := &redis.XAddArgs{
		Stream: transposedStreamID,
		Values: payload,
		MaxLen: p.maxStreamLength,
		Approx: p.approxMaxStreamLength,
		ID:     "*", // let redis create message ID
	}

	backoff := p.retry.InitialBackoff
	var lastErr error
	for attempt := 1; attempt <= p.retry.MaxAttempts; attempt++ {
		_, err := p.rdb.XAdd(ctx, args).Result()
		if err == nil {
			return nil
		}

		lastErr = err
		if !isRetryableRedisError(err) || attempt == p.retry.MaxAttempts {
			break
		}

		log.Ctx(ctx).Warn().Err(err).
			Str("stream", streamID).
			Int("attempt", attempt).
			Int("max_attempts", p.retry.MaxAttempts).
			Dur("backoff", backoff).
			Msg("transient redis produce failure, retrying")

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("failed to write to stream '%s' (redis stream '%s'): %w",
				streamID, transposedStreamID, ctx.Err())
		case <-timer.C:
		}

		next := time.Duration(float64(backoff) * p.retry.BackoffFactor)
		if next > p.retry.MaxBackoff {
			next = p.retry.MaxBackoff
		}
		backoff = next
	}

	// Only count true produce failures — not cancellation/deadline.
	if lastErr != nil &&
		!errors.Is(lastErr, context.Canceled) &&
		!errors.Is(lastErr, context.DeadlineExceeded) {
		redisProducerSendFailures.WithLabelValues(streamID).Inc()
	}

	return fmt.Errorf("failed to write to stream '%s' (redis stream '%s'). Error: %w",
		streamID, transposedStreamID, lastErr)
}

// isRetryableRedisError reports whether err is a transient Redis/network failure
// worth retrying (aligned with go-redis shouldRetry for network/LOADING/READONLY).
func isRetryableRedisError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	msg := err.Error()
	if strings.Contains(msg, "LOADING") || strings.Contains(msg, "READONLY") {
		return true
	}
	if strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connect: ") {
		return true
	}

	return false
}
