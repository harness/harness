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
	"net/http"
	"testing"
)

// TestDetectContentType locks in the behaviour of the raw endpoint's content type
// detection. The SVG special case exists because http.DetectContentType does not
// implement SVG sniffing; the CODE-5466 fix deliberately keeps serving SVG as
// image/svg+xml and relies on the sandbox CSP to prevent execution, so a change here
// would silently alter the security posture as well as the UI behaviour.
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
