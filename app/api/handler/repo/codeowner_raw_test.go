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

package repo

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/harness/gitness/app/services/codeowners"
)

const codeOwnersRawBody = "# security cares about dependency changes\n\n" +
	"package.json @security # net-new deps\n" +
	"* user1@harness.io\n"

func serveCodeOwnersRaw(t *testing.T, ifNoneMatch string) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, "/api/v1/repos/acc/org/proj/repo/+/codeowners", nil)
	if ifNoneMatch != "" {
		r.Header.Set("If-None-Match", ifNoneMatch)
	}
	w := httptest.NewRecorder()

	writeCodeOwnersRawResponse(w, r, &codeowners.File{
		Content:   codeOwnersRawBody,
		SHA:       testBlobSHA,
		TotalSize: int64(len(codeOwnersRawBody)),
	})

	return w
}

func TestCodeOwnersRawIsServedAsPlainText(t *testing.T) {
	w := serveCodeOwnersRaw(t, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Want status %d, got %d", http.StatusOK, w.Code)
	}
	if got, want := w.Header().Get("Content-Type"), contentTypeTextPlainUTF8; got != want {
		t.Errorf("Want Content-Type %q, got %q", want, got)
	}
	if got, want := w.Header().Get("Content-Length"), strconv.Itoa(len(codeOwnersRawBody)); got != want {
		t.Errorf("Want Content-Length %q, got %q", want, got)
	}
	// RFC 9110 requires an entity-tag to be DQUOTE-delimited.
	if got, want := w.Header().Get("ETag"), `"`+testBlobSHA+`"`; got != want {
		t.Errorf("Want ETag %q, got %q", want, got)
	}
	if got := w.Body.String(); got != codeOwnersRawBody {
		t.Errorf("Want body unchanged %q, got %q", codeOwnersRawBody, got)
	}
	assertSandboxed(t, w)
}

func TestCodeOwnersRawNotModifiedKeepsSandboxHeaders(t *testing.T) {
	w := serveCodeOwnersRaw(t, `"`+testBlobSHA+`"`)

	if w.Code != http.StatusNotModified {
		t.Fatalf("Want status %d, got %d", http.StatusNotModified, w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("Want empty body on 304, got %q", w.Body.String())
	}
	// RFC 9110 15.4.5: a 304 should carry the validator so caches can refresh freshness.
	if got, want := w.Header().Get("ETag"), `"`+testBlobSHA+`"`; got != want {
		t.Errorf("Want ETag %q on 304, got %q", want, got)
	}
	assertSandboxed(t, w)
}

// TestCodeOwnersRawIfNoneMatchForms covers the coupling between the quoted ETag we send and the
// If-None-Match value we compare it against: quoting the response without normalizing the request
// header would stop the 304 from ever firing.
func TestCodeOwnersRawIfNoneMatchForms(t *testing.T) {
	tests := []struct {
		name        string
		ifNoneMatch string
		wantCode    int
	}{
		{name: "quoted matching tag", ifNoneMatch: `"` + testBlobSHA + `"`, wantCode: http.StatusNotModified},
		{name: "weak matching tag", ifNoneMatch: `W/"` + testBlobSHA + `"`, wantCode: http.StatusNotModified},
		{name: "unquoted matching tag", ifNoneMatch: testBlobSHA, wantCode: http.StatusNotModified},
		{name: "wildcard", ifNoneMatch: "*", wantCode: http.StatusNotModified},
		{name: "different tag", ifNoneMatch: `"0000000000000000000000000000000000000000"`, wantCode: http.StatusOK},
		{name: "absent header", ifNoneMatch: "", wantCode: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := serveCodeOwnersRaw(t, tt.ifNoneMatch)

			if w.Code != tt.wantCode {
				t.Fatalf("Want status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}
