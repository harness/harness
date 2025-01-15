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

	"github.com/harness/gitness/types/capabilities"
)

var DisplayErrorType capabilities.Type = "display_error"
var DisplayErrorVersion capabilities.Version = "0"

type DisplayErrorInput struct {
	Error string `json:"error"`
}

func (DisplayErrorInput) IsCapabilityInput() {}

type DisplayErrorOutput struct {
	Error string `json:"error"`
}

func (DisplayErrorOutput) IsCapabilityOutput() {}

const AIContextPayloadTypeDisplayPipelineError capabilities.AIContextPayloadType = "other"

func (DisplayErrorOutput) GetType() capabilities.AIContextPayloadType {
	return AIContextPayloadTypeDisplayPipelineError
}

func (DisplayErrorOutput) GetName() string {
	return string(DisplayErrorType)
}

func (r *Registry) RegisterDisplayErrorCapability(
	logic func(ctx context.Context, input *DisplayErrorInput) (*DisplayErrorOutput, error),
) error {
	return r.register(
		capabilities.Capability{
			Type:         DisplayErrorType,
			NewInput:     func() capabilities.Input { return &DisplayErrorInput{} },
			Logic:        newLogic(logic),
			Version:      DisplayErrorVersion,
			ReturnToUser: true,
		},
	)
}

func DisplayError() func(
	ctx context.Context,
	input *DisplayErrorInput) (*DisplayErrorOutput, error) {
	return func(_ context.Context, input *DisplayErrorInput) (*DisplayErrorOutput, error) {
		return &DisplayErrorOutput{
			Error: input.Error,
		}, nil
	}
}
