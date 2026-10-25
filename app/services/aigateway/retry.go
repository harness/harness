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

package aigateway

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// maxRetries is the number of retries after the first attempt.
	maxRetries     = 2
	retryBaseDelay = time.Second
	retryMaxDelay  = 30 * time.Second
)

// withRetry runs call until it succeeds, fails with a non-retryable error, runs out of
// retries or the context is cancelled. Every attempt, retries included, reserves one request
// of the class first, because OpenRouter charges failed requests too. A quota error stops the retries.
func (g *Gateway) withRetry(ctx context.Context, class Class, call func() error) error {
	for attempt := 0; ; attempt++ {
		if err := g.planner.Reserve(class); err != nil {
			g.emitQuotaExhausted(ctx, err)
			return err
		}

		err := call()
		if err == nil || attempt >= maxRetries || !retryable(err) || ctx.Err() != nil {
			return err
		}
		if err := g.sleep(ctx, retryDelay(attempt, err)); err != nil {
			return err
		}
	}
}

// retryable reports whether OpenRouter asked us to try again later: rate limits and server errors.
func retryable(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= http.StatusInternalServerError
}

// retryDelay returns the wait before the next retry: Retry-After if the server sent one,
// otherwise exponential backoff (1s, 2s, ...). Both are capped at retryMaxDelay.
func retryDelay(attempt int, err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		return min(apiErr.RetryAfter, retryMaxDelay)
	}
	return min(retryBaseDelay<<attempt, retryMaxDelay)
}

// parseRetryAfter reads a Retry-After header given in seconds. HTTP dates are ignored.
func parseRetryAfter(value string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || secs < 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

// sleepContext waits for d or until ctx is done.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
