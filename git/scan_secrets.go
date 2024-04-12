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
	"github.com/harness/gitness/git/sharedrepo"
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

	var findings []api.Finding

	err := sharedrepo.Run(ctx, nil, s.tmpDir, repoPath, func(sharedRepo *sharedrepo.SharedRepo) error {
		var err error
		findings, err = s.git.ScanSecrets(sharedRepo.Directory(), params.BaseRev, params.Rev)
		return err
	}, params.AlternateObjectDirs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaks on diff: %w", err)
	}

	return &ScanSecretsOutput{
		Findings: findings,
	}, nil
}
