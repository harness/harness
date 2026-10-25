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

package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/controller/system"
	"github.com/harness/gitness/app/services/aigateway"
)

func TestHandleAIDisabled(t *testing.T) {
	gw := aigateway.New(aigateway.Config{}, &aigateway.Redactor{})
	ctrl := system.NewController(nil, nil, gw)

	cases := map[string]http.HandlerFunc{
		"models": HandleAIModels(ctrl),
		"quota":  HandleAIQuota(ctrl),
	}
	for name, h := range cases {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/api/v1/system/ai/"+name, nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", name, rec.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: invalid json: %v", name, err)
		}
		if body["enabled"] != false {
			t.Errorf("%s: expected enabled=false, got %v", name, body["enabled"])
		}
	}
}
