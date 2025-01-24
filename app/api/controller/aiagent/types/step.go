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

package types

import (
	"fmt"
	"strings"

	"github.com/harness/gitness/app/api/controller/aiagent/types/enum"
	"github.com/harness/gitness/app/api/usererror"
)

type PipelineStepData struct {
	Yaml string `json:"yaml_step"`
}

// create.
type GeneratePipelineStepInput struct {
	Prompt       string            `json:"prompt"`
	Metadata     map[string]string `json:"metadata"`
	Conversation []Conversation    `json:"conversation"`
}

type GeneratePipelineStepOutput struct {
	Error string           `json:"error"`
	Data  PipelineStepData `json:"data"`
}

// update.
type UpdatePipelineStepInput struct {
	Prompt       string           `json:"prompt"`
	Data         PipelineStepData `json:"data"`
	Conversation []Conversation   `json:"conversation"`
}

type UpdatePipelineStepOutput struct {
	Error string           `json:"error"`
	Data  PipelineStepData `json:"data"`
}

func (in *GeneratePipelineStepInput) Sanitize() error {
	for _, c := range in.Conversation {
		if c.Role == "" {
			return usererror.BadRequest("role must be provided")
		}

		sanitizedRole, valid := c.Role.Sanitize()
		if !valid {
			rolesStr := toString(enum.GetAllRoles)
			return usererror.BadRequestf(
				"invalid type given for Role: %s, allowed values are '%s'", c.Role, rolesStr)
		}
		c.Role = sanitizedRole

		if c.Message.Type == "" && c.Message.Data == "" {
			continue
		}

		if c.Message.Type == "" {
			return usererror.BadRequestf("message type must be provided")
		}

		sanitizedType, valid := c.Message.Type.Sanitize()
		if !valid {
			messageTypeStr := toString(enum.GetAllMessageTypes)
			return usererror.BadRequestf(
				"invalid type given for message type: %s, allowed values are %s", c.Message.Type, messageTypeStr)
		}
		c.Message.Type = sanitizedType

		if c.Message.Data == "" {
			return usererror.BadRequest("message data must be provided")
		}
	}

	return nil
}

func toString[T any](getAll func() ([]T, T)) string {
	values, _ := getAll()
	strValues := make([]string, len(values))
	for i, value := range values {
		strValues[i] = fmt.Sprintf("%v", value)
	}
	return strings.Join(strValues, ", ")
}
