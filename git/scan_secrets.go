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

	"github.com/harness/gitness/git/api"
)

type ScanSecretsParams struct {
	ReadParams

	BaseRev string // optional to scan for secrets on diff between baseRev and Rev
	Rev     string
}

type ScanSecretsOutput struct {
	Findings []api.Finding
}

func (s *Service) ScanSecrets(ctx context.Context, params *ScanSecretsParams) (*ScanSecretsOutput, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	// Create a directory for the temporary shared repository.
	shared, err := api.NewSharedRepo(s.git, s.tmpDir, params.RepoUID, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create shared repository: %w", err)
	}
	defer shared.Close(ctx)

	// Create bare repository with alternates pointing to the original repository.
	err = shared.InitAsSharedWithAlternates(ctx, params.AlternateObjectDirs...)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp repo with alternates: %w", err)
	}

	findings, err := s.git.ScanSecrets(shared.RepoPath, params.BaseRev, params.Rev)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaks on diff: %w", err)
	}

	return &ScanSecretsOutput{
		Findings: findings,
	}, nil
}
