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

package git

import (
	"context"
	"fmt"
)

type GetSubmoduleParams struct {
	ReadParams
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF string
	Path   string
}

type GetSubmoduleOutput struct {
	Submodule Submodule
}
type Submodule struct {
	Name string
	URL  string
}

func (s *Service) GetSubmodule(ctx context.Context, params *GetSubmoduleParams) (*GetSubmoduleOutput, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	// TODO: do we need to validate request for nil?
	gitSubmodule, err := s.git.GetSubmodule(ctx, repoPath, params.GitREF, params.Path)
	if err != nil {
		return nil, fmt.Errorf("GetSubmodule: failed to get submodule: %w", err)
	}

	return &GetSubmoduleOutput{
		Submodule: Submodule{
			Name: gitSubmodule.Name,
			URL:  gitSubmodule.URL,
		},
	}, nil
}
