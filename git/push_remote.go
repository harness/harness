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

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
)

type PushRemoteParams struct {
	ReadParams
	RemoteURL string
}

func (p *PushRemoteParams) Validate() error {
	if p == nil {
		return ErrNoParamsProvided
	}

	if err := p.ReadParams.Validate(); err != nil {
		return err
	}

	if p.RemoteURL == "" {
		return errors.InvalidArgument("remote url cannot be empty")
	}
	return nil
}

func (s *Service) PushRemote(ctx context.Context, params *PushRemoteParams) error {
	if err := params.Validate(); err != nil {
		return err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	isEmpty, err := s.git.HasBranches(ctx, repoPath)
	if err != nil {
		return errors.Internal(err, "push to repo failed")
	}
	if isEmpty {
		return errors.InvalidArgument("cannot push empty repo")
	}

	err = s.git.Push(ctx, repoPath, api.PushOptions{
		Remote: params.RemoteURL,
		Force:  false,
		Env:    nil,
		Mirror: true,
	})
	if err != nil {
		return fmt.Errorf("PushRemote: failed to push to remote repository: %w", err)
	}
	return nil
}
