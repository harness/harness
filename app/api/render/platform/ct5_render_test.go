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

package platform

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCt5RenderResource(t *testing.T) {
	tests := []struct {
		name       string
		code       int
		wantStatus string
	}{
		{"ok", http.StatusOK, "SUCCESS"},
		{"redirect", http.StatusFound, "FAILURE"},
		{"client error", http.StatusBadRequest, "ERROR"},
		{"server error", http.StatusInternalServerError, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			RenderResource(rec, tt.code, map[string]string{"k": "v"})

			require.Equal(t, tt.code, rec.Code)
			var out struct {
				Status string            `json:"status"`
				Data   map[string]string `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
			require.Equal(t, tt.wantStatus, out.Status)
			require.Equal(t, "v", out.Data["k"])
		})
	}
}
