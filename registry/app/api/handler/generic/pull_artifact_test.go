//  Copyright 2023 Harness, Inc.
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

package generic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/registry/app/pkg"

	"github.com/stretchr/testify/assert"
)

func TestPullArtifact_InvalidPath(t *testing.T) {
	handler := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/invalid", nil)
	w := httptest.NewRecorder()
	handler.PullArtifact(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServeContent_NilReader(t *testing.T) {
	handler := &Handler{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	info := pkg.GenericArtifactInfo{FileName: "test.txt"}

	handler.serveContent(w, req, nil, info)

	// Should not crash with nil reader
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServeContent_ValidFilename(t *testing.T) {
	handler := &Handler{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	info := pkg.GenericArtifactInfo{FileName: "test.txt"}

	handler.serveContent(w, req, nil, info)

	// Should handle the call without crashing
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServeContent_EmptyFilename(t *testing.T) {
	handler := &Handler{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	info := pkg.GenericArtifactInfo{FileName: ""}

	handler.serveContent(w, req, nil, info)

	// Should handle empty filename
	assert.Equal(t, http.StatusOK, w.Code)
}
