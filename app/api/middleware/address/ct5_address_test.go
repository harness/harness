// Copyright 2026 Harness, Inc.
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

package address

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCt5HandlerSchemeAndHost(t *testing.T) {
	tests := []struct {
		name       string
		scheme     string
		host       string
		prepare    func(r *http.Request)
		wantScheme string
		wantHost   string
	}{
		{
			name: "configured values win", scheme: "https", host: "cfg.example.com",
			wantScheme: "https", wantHost: "cfg.example.com",
		},
		{
			name:       "default scheme and request host",
			prepare:    func(r *http.Request) { r.Host = "req.example.com" },
			wantScheme: "http", wantHost: "req.example.com",
		},
		{
			name:       "tls request is https",
			prepare:    func(r *http.Request) { r.TLS = &tls.ConnectionState{} },
			wantScheme: "https", wantHost: "example.com",
		},
		{
			name:       "https proto prefix",
			prepare:    func(r *http.Request) { r.Proto = "HTTPS/1.1" },
			wantScheme: "https", wantHost: "example.com",
		},
		{
			name:       "forwarded proto header",
			prepare:    func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "https") },
			wantScheme: "https", wantHost: "example.com",
		},
		{
			name: "url host is reset before resolution",
			prepare: func(r *http.Request) {
				r.Host = ""
				r.URL.Host = "url.example.com"
			},
			wantScheme: "http", wantHost: "localhost:3000",
		},
		{
			name: "x-forwarded-for fallback",
			prepare: func(r *http.Request) {
				r.Host = ""
				r.URL.Host = ""
				r.Header.Set("X-Forwarded-For", "xff.example.com")
			},
			wantScheme: "http", wantHost: "xff.example.com",
		},
		{
			name: "x-host fallback",
			prepare: func(r *http.Request) {
				r.Host = ""
				r.URL.Host = ""
				r.Header.Set("X-Host", "xhost.example.com")
			},
			wantScheme: "http", wantHost: "xhost.example.com",
		},
		{
			name: "xff fallback",
			prepare: func(r *http.Request) {
				r.Host = ""
				r.URL.Host = ""
				r.Header.Set("XFF", "xff2.example.com")
			},
			wantScheme: "http", wantHost: "xff2.example.com",
		},
		{
			name: "x-real-ip fallback",
			prepare: func(r *http.Request) {
				r.Host = ""
				r.URL.Host = ""
				r.Header.Set("X-Real-IP", "10.0.0.1")
			},
			wantScheme: "http", wantHost: "10.0.0.1",
		},
		{
			name: "localhost default",
			prepare: func(r *http.Request) {
				r.Host = ""
				r.URL.Host = ""
			},
			wantScheme: "http", wantHost: "localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotScheme, gotHost string
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				gotScheme, gotHost = r.URL.Scheme, r.URL.Host
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.prepare != nil {
				tt.prepare(req)
			}
			Handler(tt.scheme, tt.host)(next).ServeHTTP(httptest.NewRecorder(), req)

			require.Equal(t, tt.wantScheme, gotScheme)
			require.Equal(t, tt.wantHost, gotHost)
		})
	}
}
