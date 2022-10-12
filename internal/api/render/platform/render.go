// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package platform

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/render"
)

// RenderResource is a helper function that renders a single
// resource, wrapped in the harness payload envelope.
func RenderResource(w http.ResponseWriter, code int, v interface{}) {
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
