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

package aiagent

import (
	"context"
	"fmt"

	controllertypes "github.com/harness/gitness/app/api/controller/aiagent/types"
	aitypes "github.com/harness/gitness/types/aigenerate"
)

func (c *Controller) GeneratePipeline(
	ctx context.Context,
	in *controllertypes.GeneratePipelineInput,
) (*controllertypes.GeneratePipelineOutput, error) {
	generateRequest := &aitypes.PipelineGenerateRequest{
		Prompt:  in.Prompt,
		RepoRef: in.RepoRef,
	}

	output, err := c.intelligence.GeneratePipeline(ctx, generateRequest)
	if err != nil {
		return nil, fmt.Errorf("generate pipeline: %w", err)
	}
	return &controllertypes.GeneratePipelineOutput{
		Status: "SUCCESS",
		Data: controllertypes.PipelineData{
			YamlPipeline: output.YAML,
		},
	}, nil
}
