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

package repo

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/git/sha"
)

// synackSVGPayload is the base64 body from the CODE-6295 Synack report (raven-w001-102). It is
// an SVG whose <script> calls alert(document.domain); committed to a repo and opened through the
// raw endpoint it used to execute on the application origin.
const synackSVGPayload = "PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjxzY3JpcHQ+" +
	"YWxlcnQoZG9jdW1lbnQuZG9tYWluKTwvc2NyaXB0Pjwvc3ZnPg=="

const testBlobSHA = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// serveRaw runs the raw endpoint's response path over the given blob bytes. It exercises
// writeRawResponse rather than HandleRaw so no repo controller, git service or auth session is
// needed - everything CODE-6295 is about happens after the blob has been read.
func serveRaw(t *testing.T, filePath string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, "/api/v1/repos/acc/org/proj/repo/+/raw/"+filePath, nil)
	w := httptest.NewRecorder()

	writeRawResponse(w, r, &repo.RawContent{
		Data: io.NopCloser(strings.NewReader(string(body))),
		Size: int64(len(body)),
		SHA:  sha.Must(testBlobSHA),
	})

	return w
}

// assertSandboxed checks the two headers that stop user-controlled bytes from becoming an
// executable document on the application origin.
func assertSandboxed(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	if got, want := w.Header().Get("X-Content-Type-Options"), "nosniff"; got != want {
		t.Errorf("Want X-Content-Type-Options %q, got %q", want, got)
	}

	csp := w.Header().Get("Content-Security-Policy")
	for _, directive := range []string{"default-src 'none'", "sandbox", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, directive) {
			t.Errorf("Want Content-Security-Policy to contain %q, got %q", directive, csp)
		}
	}
}

// TestRawSynackSVGPayloadIsNeutralized reproduces CODE-6295 step 5: fetch the committed xss.svg
// through the raw endpoint, the same way the Synack report and Ryan Garbars' retest opened it.
// Per the Synack recommended fix's text/plain option, image/svg+xml is rewritten to text/plain
// unconditionally, the same as text/html - the response still carries the SVG bytes verbatim
// (the raw endpoint's job is to return the file unchanged, and "View Raw" must still show it),
// but the browser displays them as text instead of parsing them as a document.
func TestRawSynackSVGPayloadIsNeutralized(t *testing.T) {
	body, err := base64.StdEncoding.DecodeString(synackSVGPayload)
	if err != nil {
		t.Fatalf("Failed to decode the Synack payload: %v", err)
	}

	w := serveRaw(t, "xss.svg", body)

	if got, want := w.Code, http.StatusOK; got != want {
		t.Fatalf("Want status %d, got %d", want, got)
	}
	if got, want := w.Body.String(), string(body); got != want {
		t.Errorf("Want body %q, got %q", want, got)
	}

	assertSandboxed(t, w)

	if got, want := w.Header().Get("Content-Type"), "text/plain; charset=utf-8"; got != want {
		t.Errorf("Want Content-Type %q so the browser shows the markup as text, got %q", want, got)
	}

	if got := w.Header().Get("Content-Disposition"); got != "" {
		t.Errorf("Want no Content-Disposition so View Raw keeps showing the file, got %q", got)
	}
}

// TestRawHTMLIsForcedToPlainText covers the other half of the CODE-6295 report: an HTML file
// served inline from the app origin. It is rewritten to text/plain so the browser shows the
// source instead of parsing it as a document, and it is never downloaded.
func TestRawHTMLIsForcedToPlainText(t *testing.T) {
	body := []byte(`<html><body><script>alert(document.domain)</script></body></html>`)

	w := serveRaw(t, "docs/evil.html", body)

	assertSandboxed(t, w)

	if got, want := w.Header().Get("Content-Type"), "text/plain; charset=utf-8"; got != want {
		t.Errorf("Want Content-Type %q, got %q", want, got)
	}
	if got, want := w.Body.String(), string(body); got != want {
		t.Errorf("Want body %q, got %q", want, got)
	}
	if got := w.Header().Get("Content-Disposition"); got != "" {
		t.Errorf("Want no Content-Disposition, got %q", got)
	}
}

// TestRawTextStaysInline pins the behaviour the forced download must not regress: "View Raw" on
// an ordinary source file has to keep rendering in the browser instead of downloading.
func TestRawTextStaysInline(t *testing.T) {
	for _, test := range []struct {
		name     string
		filePath string
		body     string
	}{
		{name: "source file", filePath: "main.go", body: "package main\n\nfunc main() {}\n"},
		{name: "png image", filePath: "logo.png", body: "\x89PNG\r\n\x1a\nbinary payload"},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := serveRaw(t, test.filePath, []byte(test.body))

			assertSandboxed(t, w)

			if got := w.Header().Get("Content-Disposition"); got != "" {
				t.Errorf("Want no Content-Disposition for %s, got %q", test.filePath, got)
			}
		})
	}
}

// TestRawNotModifiedKeepsSecurityHeaders makes sure a revalidated response cannot be served
// without the sandbox policy. The headers used to be set after the If-None-Match short circuit.
func TestRawNotModifiedKeepsSecurityHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/repos/acc/org/proj/repo/+/raw/xss.svg", nil)
	r.Header.Set(request.HeaderIfNoneMatch, testBlobSHA)
	w := httptest.NewRecorder()

	writeRawResponse(w, r, &repo.RawContent{
		Data: io.NopCloser(strings.NewReader("ignored")),
		Size: 7,
		SHA:  sha.Must(testBlobSHA),
	})

	if got, want := w.Code, http.StatusNotModified; got != want {
		t.Fatalf("Want status %d, got %d", want, got)
	}

	assertSandboxed(t, w)
}

// TestDetectContentType locks in the behaviour of the raw endpoint's content type
// detection. The SVG special case exists because http.DetectContentType does not
// implement SVG sniffing; detecting image/svg+xml here is what lets
// render.NeutralizeRenderableContentType recognize it and rewrite it to text/plain
// unconditionally, per the CODE-6295 fix, so a change here would silently alter the
// security posture as well as the UI behaviour.
func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "svg with onload attribute",
			data: `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><circle r="8"/></svg>`,
			want: "image/svg+xml",
		},
		{
			name: "svg behind xml declaration",
			data: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`,
			want: "image/svg+xml",
		},
		{
			name: "svg with uppercase tag",
			data: `<SVG xmlns="http://www.w3.org/2000/svg"><circle r="8"/></SVG>`,
			want: "image/svg+xml",
		},
		{
			name: "benign svg",
			data: `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><circle r="30"/></svg>`,
			want: "image/svg+xml",
		},
		{
			name: "xml that is not svg stays xml",
			data: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<note><body>hello</body></note>`,
			want: "text/xml; charset=utf-8",
		},
		{
			name: "html falls through to the standard sniffer",
			data: `<html><body><script>alert(1)</script></body></html>`,
			want: "text/html; charset=utf-8",
		},
		{
			name: "png magic bytes",
			data: "\x89PNG\r\n\x1a\n" + "some binary payload here",
			want: "image/png",
		},
		{
			name: "plain text",
			data: "just a regular file with some words in it",
			want: "text/plain; charset=utf-8",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := detectContentType([]byte(test.data)); got != test.want {
				t.Errorf("Want content type %q, got %q", test.want, got)
			}
		})
	}
}

// TestDetectContentTypeShortInput makes sure the length guard does not panic and defers
// to the standard sniffer for inputs too short to match the SVG prefixes.
func TestDetectContentTypeShortInput(t *testing.T) {
	for _, data := range []string{"", "<", "<svg"} {
		want := http.DetectContentType([]byte(data))
		if got := detectContentType([]byte(data)); got != want {
			t.Errorf("Want content type %q for %q, got %q", want, data, got)
		}
	}
}
