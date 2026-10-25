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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxResponseBytes = 8 << 20

// APIError is a non-2xx response from OpenRouter.
type APIError struct {
	StatusCode int
	Body       string
	// RetryAfter is the delay from the Retry-After header in seconds, zero if absent.
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	return fmt.Sprintf("openrouter returned status %d: %s", e.StatusCode, e.Body)
}

// IsRateLimited reports whether OpenRouter rejected the request because of rate limits.
func IsRateLimited(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests
}

// httpClient is the thin transport shared by the catalog, decisions and chat clients.
type httpClient struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

func (c *httpClient) doJSON(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		const maxErrBody = 512
		if len(data) > maxErrBody {
			data = data[:maxErrBody]
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(data),
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}

	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to decode response from %s: %w", path, err)
	}
	return nil
}
