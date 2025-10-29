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

package conformanceutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

// Client represents a Maven registry client.
type Client struct {
	baseURL string
	token   string
	client  *http.Client
	debug   bool
}

// NewClient creates a new Maven registry client.
func NewClient(baseURL, token string, debug bool) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{},
		debug:   debug,
	}
}

// Request represents an HTTP request.
type Request struct {
	method  string
	path    string
	headers map[string]string
	body    any
}

// Response represents an HTTP response.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// NewRequest creates a new request.
func (c *Client) NewRequest(method, urlPath string) *Request {
	return &Request{
		method:  method,
		path:    urlPath,
		headers: make(map[string]string),
	}
}

// SetHeader sets a request header.
func (r *Request) SetHeader(key, value string) {
	r.headers[key] = value
}

// SetBody sets the request body.
func (r *Request) SetBody(body any) {
	r.body = body
}

type ReaderFile struct {
	*bytes.Buffer
}

func (r *ReaderFile) Close() error {
	return nil
}

func NewReaderFile(data []byte) *ReaderFile {
	return &ReaderFile{bytes.NewBuffer(data)}
}

func (c *Client) SetBody(req *http.Request, body []byte) {
	req.Body = io.NopCloser(NewReaderFile(body))
	req.ContentLength = int64(len(body))
}

func (c *Client) SetHeader(req *http.Request, key, value string) {
	req.Header.Set(key, value)
}

// Do executes the request.
func (c *Client) Do(req *Request) (*Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	cleanPath := path.Clean(req.path)
	if len(cleanPath) > 0 && cleanPath[0] == '/' {
		cleanPath = cleanPath[1:]
	}

	// Special handling for Gitness Maven registry paths
	// Gitness may require additional path components or structure
	u.Path = path.Join(u.Path, cleanPath)

	var bodyReader io.Reader
	if req.body != nil {
		var bodyBytes []byte
		switch v := req.body.(type) {
		case []byte:
			bodyBytes = v
		case string:
			bodyBytes = []byte(v)
		default:
			bodyBytes, err = json.Marshal(req.body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(context.Background(), req.method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Only log details if DEBUG is enabled
	if c.debug {
		log.Printf("Making %s request to: %s\n", httpReq.Method, httpReq.URL)
		if httpReq.Body != nil {
			log.Printf("Request Body: %v\n", httpReq.Body)
		}
		for k, v := range httpReq.Header {
			log.Printf("Request Header: %s: %v\n", k, v)
		}
	}

	// Set authorization header with PAT token for Gitness authentication.
	if c.token != "" {
		// Always use Bearer token authentication for Gitness
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
		if c.debug {
			log.Printf("Using Gitness Bearer token for authentication\n")
		}
	} else {
		// Always log authentication warnings regardless of debug setting
		log.Printf("WARNING: No authentication token provided\n")
	}

	// Set headers.
	for k, v := range req.headers {
		httpReq.Header.Set(k, v)
	}

	// Set default content-type if not set.
	if req.body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		// Always log critical errors
		log.Printf("ERROR: Failed to execute request: %v\n", err)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Print response details only if DEBUG is enabled
	if c.debug {
		log.Printf("Response Status: %d %s\n", resp.StatusCode, resp.Status)
		for k, v := range resp.Header {
			log.Printf("Response Header: %s: %v\n", k, v)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Always log critical errors
		log.Printf("ERROR: Failed to read response body: %v\n", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) > 0 {
		log.Printf("Response Body: %s\n", string(body))
	}

	if resp.StatusCode >= 400 {
		log.Printf("Error response: %d %s\n", resp.StatusCode, resp.Status)
		log.Printf("Error response body: %s\n", string(body))
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}, nil
}
