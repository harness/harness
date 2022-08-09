package platform

import (
	"encoding/json"
	"net/http"

	"github.com/bradrydzewski/my-app/internal/api/render"
)

// RenderResource is a helper function that renders a single
// resource, wrapped in the harness payload envelope.
func RenderResource(w http.ResponseWriter, v interface{}, code int) {
	payload := new(wrapper)
	payload.Status = "SUCCESS"
	payload.Data, _ = json.Marshal(v)
	if code > 399 {
		payload.Status = "ERROR"
	} else if code > 299 {
		payload.Status = "FAILURE"
	}
	render.JSON(w, payload, code)
}

// wrapper defines the payload wrapper.
type wrapper struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}
