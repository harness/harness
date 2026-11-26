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

package requestutil

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRrmRemoteAddrForwardedFor(t *testing.T) {
	req := httptestNewRequest(t)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.2")
	require.Equal(t, "203.0.113.1", RemoteAddr(req))
}

func TestRrmRemoteAddrRealIP(t *testing.T) {
	req := httptestNewRequest(t)
	req.Header.Set("X-Real-Ip", "198.51.100.9")
	require.Equal(t, "198.51.100.9", RemoteAddr(req))
}

func TestRrmRemoteIPStripsPort(t *testing.T) {
	req := httptestNewRequest(t)
	req.RemoteAddr = "10.0.0.5:1234"
	require.Equal(t, "10.0.0.5", RemoteIP(req))
}

func httptestNewRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/", nil)
	require.NoError(t, err)
	return req
}
