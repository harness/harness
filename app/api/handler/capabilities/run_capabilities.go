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

package capabilities

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/harness/gitness/app/api/controller/capabilities"
	"github.com/harness/gitness/app/api/render"
	capabilitiesservice "github.com/harness/gitness/app/services/capabilities"
	capabilities2 "github.com/harness/gitness/types/capabilities"
)

func HandleRunCapabilities(capabilitiesCtrl *capabilities.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		data, err := io.ReadAll(r.Body)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		in, err := UnmarshalRunCapabilitiesRequest(capabilitiesCtrl.Capabilities, data)
		if err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		resp, err := capabilitiesCtrl.RunCapabilities(ctx, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, resp)
	}
}

// UnmarshalRunCapabilitiesRequest We need this method in order to unmarshall
// the request. We cannot do this in a generic way, because we use the
// capability type in order to deserialize it appropriately (see DeserializeInput,
// as used in this function).
func UnmarshalRunCapabilitiesRequest(cr *capabilitiesservice.Registry,
	data []byte) (*capabilities.RunCapabilitiesRequest, error) {
	r := &capabilities.RunCapabilitiesRequest{}
	type rawCapability struct {
		CallID string             `json:"call_id"`
		Type   capabilities2.Type `json:"type"`
		Input  json.RawMessage
	}
	var rawBase struct {
		ConversationRaw   string                 `json:"conversation_raw"`
		ConversationID    capabilities.ContextID `json:"conversation_id"`
		CapabilitiesToRun []rawCapability        `json:"capabilities_to_run"`
	}
	err := json.Unmarshal(data, &rawBase)
	if err != nil {
		return nil, err
	}
	r.CapabilitiesToRun = make([]capabilities.CapabilityRunRequest, 0)

	r.ConversationRaw = rawBase.ConversationRaw
	r.ConversationID = rawBase.ConversationID
	for _, raw := range rawBase.CapabilitiesToRun {
		capabilityRequest := new(capabilities.CapabilityRunRequest)
		capabilityRequest.CallID = raw.CallID
		capabilityRequest.Type = raw.Type
		capabilityRequest.Input, err = capabilitiesservice.DeserializeInput(cr, raw.Type, raw.Input)
		if err != nil {
			return nil, err
		}
		r.CapabilitiesToRun = append(r.CapabilitiesToRun, *capabilityRequest)
	}
	return r, nil
}
