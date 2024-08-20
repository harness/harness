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

	"github.com/harness/gitness/types"
)

type SuggestPipelineInput struct {
	RepoRef  string `json:"repo_ref"`
	Pipeline string `json:"pipeline"`
}

func (c *Controller) SuggestPipeline(
	ctx context.Context,
	in *SuggestPipelineInput,
) (*types.PipelineSuggestionsResponse, error) {
	suggestionRequest := &types.PipelineSuggestionsRequest{
		RepoRef:  in.RepoRef,
		Pipeline: in.Pipeline,
	}
	output, err := c.pipeline.Suggest(ctx, suggestionRequest)
	if err != nil {
		return nil, fmt.Errorf("suggest pipeline: %w", err)
	}
	return output, nil
}
