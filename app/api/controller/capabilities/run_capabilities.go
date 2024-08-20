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
	"context"
	"fmt"

	"github.com/harness/gitness/types/capabilities"
)

type ContextID string

type RunCapabilitiesRequest struct {
	ConversationRaw   string                 `json:"conversation_raw"`
	ConversationID    ContextID              `json:"conversation_id"`
	CapabilitiesToRun []CapabilityRunRequest `json:"capabilities_to_run"`
}

type CapabilityRunRequest struct {
	CallID string             `json:"call_id"`
	Type   capabilities.Type  `json:"type"`
	Input  capabilities.Input `json:"input"`
}

type CapabilityExecution struct {
	Type         capabilities.Type   `json:"capability_id"`
	Result       capabilities.Output `json:"result"`
	ReturnToUser bool                `json:"return_to_user"`
}

func (c CapabilityExecution) GetType() capabilities.AIContextPayloadType {
	return "other"
}

type CapabilityRunResponse struct {
	CapabilitiesRan []CapabilityExecution `json:"capabilities_ran"`
}

func (c *Controller) RunCapabilities(ctx context.Context, req *RunCapabilitiesRequest) (*CapabilityRunResponse, error) {
	capOut := new(CapabilityRunResponse)
	capOut.CapabilitiesRan = []CapabilityExecution{}

	for _, value := range req.CapabilitiesToRun {
		if !c.Capabilities.Exists(value.Type) {
			return nil, fmt.Errorf("capability %s does not exist", value.Type)
		}

		resp, err := c.Capabilities.Execute(ctx, value.Type, value.Input)
		if err != nil {
			return nil, err
		}

		returnToUser, err := c.Capabilities.ReturnToUser(value.Type)
		if err != nil {
			return nil, err
		}
		capOut.CapabilitiesRan = append(capOut.CapabilitiesRan, CapabilityExecution{
			Type:         value.Type,
			Result:       resp,
			ReturnToUser: returnToUser,
		})
	}
	return capOut, nil
}
