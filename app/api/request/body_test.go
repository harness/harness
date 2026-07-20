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

package request_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
)

const magicNumber = 42
const magicJSON = `{"number": 42}`

type dummy struct {
	Number int `json:"number"`
}

func TestDecodeBody(t *testing.T) {
	var d dummy

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := request.DecodeBody(r, &d)
		if err != nil {
			t.Fatalf("failed to decode http request body: %s", err.Error())
			return
		}

		if d.Number != magicNumber {
			t.Errorf("expected number %d, got %d", magicNumber, d.Number)
		}

		w.WriteHeader(http.StatusOK)
	}))

	defer s.Close()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		s.URL,
		bytes.NewBuffer([]byte(magicJSON)),
	)
	if err != nil {
		t.Fatalf("failed to build request: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %s", err.Error())
	}

	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	if resp == nil {
		t.Error("expected a response but got nil")
		return
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

// TestDecodeBodyPreservesRequestContext verifies that DecodeBody's deferred
// drain-and-close does not detach the server-side request context from the
// client-side cancellation.
func TestDecodeBodyPreservesRequestContext(t *testing.T) {
	handlerDecoded := make(chan struct{})
	handlerDone := make(chan struct{})

	var d dummy

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)

		if err := request.DecodeBody(r, &d); err != nil {
			t.Errorf("failed to decode http request body: %s", err.Error())
			close(handlerDecoded)
			return
		}
		close(handlerDecoded)

		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
			t.Errorf("request context was not cancelled after client cancel")
		}
	}))
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.URL,
		strings.NewReader(`{"number":42}`),
	)
	req.Header.Set("Content-Type", "application/json")

	go func() {
		resp, _ := http.DefaultClient.Do(req)
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	<-handlerDecoded
	cancel()
	<-handlerDone

	if d.Number != magicNumber {
		t.Errorf("expected number %d, got %d", magicNumber, d.Number)
	}
}

func TestDecodeBodyErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed json", body: `{"number":`},
		{name: "empty body", body: ``},
		{name: "wrong type", body: `{"number": "not-a-number"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var handlerErr error
			var d dummy

			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerErr = request.DecodeBody(r, &d)
				w.WriteHeader(http.StatusOK)
			}))
			defer s.Close()

			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				s.URL,
				strings.NewReader(tc.body),
			)
			if err != nil {
				t.Fatalf("failed to build request: %s", err.Error())
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := s.Client().Do(req)
			if err != nil {
				t.Fatalf("failed to send request: %s", err.Error())
			}
			_ = resp.Body.Close()

			if handlerErr == nil {
				t.Fatal("expected an error from DecodeBody, got nil")
			}
		})
	}
}

func TestDecodeBodyClosesBody(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader(magicJSON)}
	req := httptest.NewRequest(http.MethodPost, "/", body)

	var d dummy
	if err := request.DecodeBody(req, &d); err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if !body.closed {
		t.Error("expected DecodeBody to close the request body")
	}
}

func TestDecodeBodyTooLarge(t *testing.T) {
	// Build a JSON payload whose serialized form exceeds MaxBodySize.
	var buf bytes.Buffer
	buf.WriteString(`{"number":42,"filler":"`)
	for buf.Len() <= request.MaxBodySize {
		buf.WriteString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	}
	buf.WriteString(`"}`)

	req := httptest.NewRequest(http.MethodPost, "/", &buf)

	var d dummy
	err := request.DecodeBody(req, &d)
	if err == nil {
		t.Fatal("expected an error for oversized body, got nil")
	}
	if !errors.Is(err, request.ErrBodyTooLarge) {
		t.Errorf("expected ErrBodyTooLarge, got: %s", err)
	}
}

func TestDecodeBodyAtLimit(t *testing.T) {
	// Build a valid JSON payload no larger than MaxBodySize.
	var buf bytes.Buffer
	buf.WriteString(`{"number":42,"filler":"`)
	for buf.Len() < request.MaxBodySize-4 {
		buf.WriteByte('a')
	}
	buf.WriteString(`"}`)
	if buf.Len() > request.MaxBodySize {
		t.Fatalf("test payload is %d bytes, want <= %d", buf.Len(), request.MaxBodySize)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &buf)

	var d dummy
	if err := request.DecodeBody(req, &d); err != nil {
		t.Fatalf("unexpected error at limit: %s", err)
	}
	if d.Number != magicNumber {
		t.Errorf("expected number %d, got %d", magicNumber, d.Number)
	}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}
