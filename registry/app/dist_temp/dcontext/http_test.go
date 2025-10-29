// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dcontext

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestWithRequest(t *testing.T) {
	var req http.Request

	start := time.Now()
	req.Method = http.MethodGet
	req.Host = "example.com"
	req.RequestURI = "/test-test"
	req.Header = make(http.Header)
	req.Header.Set("Referer", "foo.com/referer")
	req.Header.Set("User-Agent", "test/0.1")

	ctx := WithRequest(Background(), &req)
	for _, tc := range []struct {
		key      string
		expected any
	}{
		{
			key:      "http.request",
			expected: &req,
		},
		{
			key: "http.request.id",
		},
		{
			key:      "http.request.method",
			expected: req.Method,
		},
		{
			key:      "http.request.host",
			expected: req.Host,
		},
		{
			key:      "http.request.uri",
			expected: req.RequestURI,
		},
		{
			key:      "http.request.referer",
			expected: req.Referer(),
		},
		{
			key:      "http.request.useragent",
			expected: req.UserAgent(),
		},
		{
			key:      "http.request.remoteaddr",
			expected: req.RemoteAddr,
		},
		{
			key: "http.request.startedat",
		},
	} {
		v := ctx.Value(tc.key)

		if v == nil {
			t.Fatalf("value not found for %q", tc.key)
		}

		if tc.expected != nil && v != tc.expected {
			t.Fatalf("%s: %v != %v", tc.key, v, tc.expected)
		}

		// Key specific checks!
		switch tc.key {
		case "http.request.id":
			if _, ok := v.(string); !ok {
				t.Fatalf("request id not a string: %v", v)
			}
		case "http.request.startedat":
			vt, ok := v.(time.Time)
			if !ok {
				t.Fatalf("value not a time: %v", v)
			}

			now := time.Now()
			if vt.After(now) {
				t.Fatalf("time generated too late: %v > %v", vt, now)
			}

			if vt.Before(start) {
				t.Fatalf("time generated too early: %v < %v", vt, start)
			}
		}
	}
}

type testResponseWriter struct {
	flushed bool
	status  int
	written int64
	header  http.Header
}

func (trw *testResponseWriter) Header() http.Header {
	if trw.header == nil {
		trw.header = make(http.Header)
	}

	return trw.header
}

func (trw *testResponseWriter) Write(p []byte) (n int, err error) {
	if trw.status == 0 {
		trw.status = http.StatusOK
	}

	n = len(p)
	trw.written += int64(n)
	return
}

func (trw *testResponseWriter) WriteHeader(status int) {
	trw.status = status
}

func (trw *testResponseWriter) Flush() {
	trw.flushed = true
}

func TestWithResponseWriter(t *testing.T) {
	trw := testResponseWriter{}
	ctx, rw := WithResponseWriter(Background(), &trw)

	if ctx.Value("http.response") != rw {
		t.Fatalf("response not available in context: %v != %v", ctx.Value("http.response"), rw)
	}

	grw, err := GetResponseWriter(ctx)
	if err != nil {
		t.Fatalf("error getting response writer: %v", err)
	}

	if grw != rw {
		t.Fatalf("unexpected response writer returned: %#v != %#v", grw, rw)
	}

	if ctx.Value("http.response.status") != 0 {
		t.Fatalf(
			"response status should always be a number and should be zero here: %v != 0",
			ctx.Value("http.response.status"),
		)
	}

	if n, err := rw.Write(make([]byte, 1024)); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	} else if n != 1024 {
		t.Fatalf("unexpected number of bytes written: %v != %v", n, 1024)
	}

	if ctx.Value("http.response.status") != http.StatusOK {
		t.Fatalf("unexpected response status in context: %v != %v", ctx.Value("http.response.status"), http.StatusOK)
	}

	if ctx.Value("http.response.written") != int64(1024) {
		t.Fatalf("unexpected number reported bytes written: %v != %v", ctx.Value("http.response.written"), 1024)
	}

	// Make sure flush propagates
	rw.(http.Flusher).Flush() //nolint:errcheck

	if !trw.flushed {
		t.Fatalf("response writer not flushed")
	}

	// Write another status and make sure context is correct. This normally
	// wouldn't work except for in this contrived testcase.
	rw.WriteHeader(http.StatusBadRequest)

	if ctx.Value("http.response.status") != http.StatusBadRequest {
		t.Fatalf(
			"unexpected response status in context: %v != %v",
			ctx.Value("http.response.status"),
			http.StatusBadRequest,
		)
	}
}

func TestWithVars(t *testing.T) {
	var req http.Request
	vars := map[string]string{
		"foo": "asdf",
		"bar": "qwer",
	}

	getVarsFromRequest = func(r *http.Request) map[string]string {
		if r != &req {
			t.Fatalf("unexpected request: %v != %v", r, req)
		}

		return vars
	}

	ctx := WithVars(Background(), &req)
	for _, tc := range []struct {
		key      string
		expected any
	}{
		{
			key:      "vars",
			expected: vars,
		},
		{
			key:      "vars.foo",
			expected: "asdf",
		},
		{
			key:      "vars.bar",
			expected: "qwer",
		},
	} {
		v := ctx.Value(tc.key)

		if !reflect.DeepEqual(v, tc.expected) {
			t.Fatalf("%q: %v != %v", tc.key, v, tc.expected)
		}
	}
}
