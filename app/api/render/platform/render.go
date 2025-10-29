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

package platform

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/app/api/render"
)

// RenderResource is a helper function that renders a single
// resource, wrapped in the harness payload envelope.
func RenderResource(w http.ResponseWriter, code int, v any) {
	payload := new(wrapper)
	payload.Status = "SUCCESS"
	payload.Data, _ = json.Marshal(v)
	if code >= http.StatusBadRequest {
		payload.Status = "ERROR"
	} else if code >= http.StatusMultipleChoices {
		payload.Status = "FAILURE"
	}
	render.JSON(w, code, payload)
}

// wrapper defines the payload wrapper.
type wrapper struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}
