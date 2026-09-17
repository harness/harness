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

// TestNeutralizeRenderableContentType covers which media types the CODE-6295 fix rewrites to
// text/plain, per the Synack recommended fix's second option: force text/html, image/svg+xml and
// application/xhtml+xml - and anything else a browser renders as a document - to text/plain
// unconditionally, regardless of how the request reached the raw endpoint. Unlike forcing a
// download, this keeps "View Raw" actually showing the file: the browser displays the literal
// markup as text instead of parsing it as a document, so no script can run.
func TestNeutralizeRenderableContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{name: "html", contentType: "text/html; charset=utf-8", want: "text/plain; charset=utf-8"},
		{name: "html without parameters", contentType: "text/html", want: "text/plain; charset=utf-8"},
		{name: "xhtml", contentType: "application/xhtml+xml", want: "text/plain; charset=utf-8"},
		{name: "text xml", contentType: "text/xml; charset=utf-8", want: "text/plain; charset=utf-8"},
		{name: "application xml", contentType: "application/xml", want: "text/plain; charset=utf-8"},
		{name: "xsl", contentType: "text/xsl", want: "text/plain; charset=utf-8"},
		{
			name:        "xslt can script through a stylesheet PI",
			contentType: "application/xslt+xml",
			want:        "text/plain; charset=utf-8",
		},
		{name: "uppercase is still html", contentType: "TEXT/HTML; CHARSET=UTF-8", want: "text/plain; charset=utf-8"},
		{name: "padded media type", contentType: "  text/html ; charset=utf-8", want: "text/plain; charset=utf-8"},

		// SVG is explicitly named in the Synack recommended fix and is neutralized
		// unconditionally, the same as HTML - there is no <img src> exemption.
		{name: "svg is forced to plain text", contentType: "image/svg+xml", want: "text/plain; charset=utf-8"},
		{name: "svg uppercase is forced to plain text", contentType: "IMAGE/SVG+XML", want: "text/plain; charset=utf-8"},

		{name: "plain text is unchanged", contentType: "text/plain; charset=utf-8", want: "text/plain; charset=utf-8"},
		{name: "png is unchanged", contentType: "image/png", want: "image/png"},
		{name: "pdf is unchanged", contentType: "application/pdf", want: "application/pdf"},
		{name: "octet stream is unchanged", contentType: "application/octet-stream", want: "application/octet-stream"},
		{name: "empty content type is unchanged", contentType: "", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NeutralizeRenderableContentType(test.contentType)

			if got != test.want {
				t.Errorf("Want NeutralizeRenderableContentType(%q) = %q, got %q", test.contentType, test.want, got)
			}
		})
	}
}

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
