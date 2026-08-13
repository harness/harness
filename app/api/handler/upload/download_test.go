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

package upload

import (
	"strings"
	"testing"
)

// TestContentTypeForFile checks that the download handler declares an explicit content
// type instead of leaving it to the standard library's sniffer. Uploaded files are
// stored as "<uuid><extension>" where the extension was derived from the sniffed mime
// type at upload time.
func TestContentTypeForFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "svg keeps its declared type",
			filename: "7b1f2c9e-0a4d-4a1b-9c3e-2f6d8a5b1c40.svg",
			want:     "image/svg+xml",
		},
		{
			name:     "png",
			filename: "7b1f2c9e-0a4d-4a1b-9c3e-2f6d8a5b1c40.png",
			want:     "image/png",
		},
		{
			name:     "no extension falls back to octet-stream",
			filename: "7b1f2c9e-0a4d-4a1b-9c3e-2f6d8a5b1c40",
			want:     defaultContentType,
		},
		{
			name:     "unknown extension falls back to octet-stream",
			filename: "7b1f2c9e-0a4d-4a1b-9c3e-2f6d8a5b1c40.notarealextension",
			want:     defaultContentType,
		},
		{
			name:     "empty filename falls back to octet-stream",
			filename: "",
			want:     defaultContentType,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := contentTypeForFile(test.filename)
			// mime.TypeByExtension can append a charset depending on the host's mime
			// database, so compare on the media type only.
			if mediaType, _, _ := strings.Cut(got, ";"); strings.TrimSpace(mediaType) != test.want {
				t.Errorf("Want content type %q for %q, got %q", test.want, test.filename, got)
			}
		})
	}
}
