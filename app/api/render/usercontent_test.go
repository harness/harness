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

package render

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserContentSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()

	UserContentSecurityHeaders(w)

	if got, want := w.Header().Get("X-Content-Type-Options"), "nosniff"; got != want {
		t.Errorf("Want X-Content-Type-Options %q, got %q", want, got)
	}

	want := "default-src 'none'; style-src 'unsafe-inline'; sandbox; frame-ancestors 'none'"
	if got := w.Header().Get("Content-Security-Policy"); got != want {
		t.Errorf("Want Content-Security-Policy %q, got %q", want, got)
	}
}

// TestUserContentSecurityHeadersDirectives guards the individual directives that the
// CODE-5466 fix depends on, so that a future reword of the policy string cannot silently
// drop one of them.
func TestUserContentSecurityHeadersDirectives(t *testing.T) {
	w := httptest.NewRecorder()

	UserContentSecurityHeaders(w)

	csp := w.Header().Get("Content-Security-Policy")
	for _, directive := range []string{"default-src 'none'", "sandbox", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, directive) {
			t.Errorf("Want Content-Security-Policy to contain %q, got %q", directive, csp)
		}
	}
}

// TestUserContentSecurityHeadersOverwrites makes sure a previously set header cannot
// weaken the response, since some handlers set headers before calling the helper.
func TestUserContentSecurityHeadersOverwrites(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("X-Content-Type-Options", "")
	w.Header().Set("Content-Security-Policy", "default-src *")

	UserContentSecurityHeaders(w)

	if got, want := w.Header().Get("X-Content-Type-Options"), "nosniff"; got != want {
		t.Errorf("Want X-Content-Type-Options %q, got %q", want, got)
	}
	if got := w.Header().Get("Content-Security-Policy"); !strings.Contains(got, "sandbox") {
		t.Errorf("Want Content-Security-Policy to contain sandbox, got %q", got)
	}
}
