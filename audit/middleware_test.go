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

package audit

import (
	"net/http/httptest"
	"testing"
)

func TestRealIPFromIPv6RemoteAddress(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "[2001:db8::1]:1234"

	if got, want := RealIP(request), "2001:db8::1"; got != want {
		t.Fatalf("RealIP() = %q, want %q", got, want)
	}
}

func TestRealIPTrimsForwardedAddress(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(xForwardedFor, " 192.0.2.1 , 198.51.100.1")

	if got, want := RealIP(request), "192.0.2.1"; got != want {
		t.Fatalf("RealIP() = %q, want %q", got, want)
	}
}
