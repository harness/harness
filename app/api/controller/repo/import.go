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

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
)

type ImportInput struct {
	ParentRef string `json:"parent_ref"`
	// TODO [CODE-1363]: remove after identifier migration.
	UID         string `json:"uid" deprecated:"true"`
	Identifier  string `json:"identifier"`
	Description string `json:"description"`

	Provider     importer.Provider `json:"provider"`
	ProviderRepo string            `json:"provider_repo"`

	Pipelines importer.PipelineOption `json:"pipelines"`
}

// Import creates a new empty repository and starts git import to it from a remote repository.
func (c *Controller) Import(ctx context.Context, session *auth.Session, in *ImportInput) (*types.Repository, error) {
	if err := c.sanitizeImportInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	parentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	var repo *types.Repository
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.resourceLimiter.RepoCount(ctx, parentSpace.ID, 1); err != nil {
			return fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
		}

		remoteRepository, provider, err := importer.LoadRepositoryFromProvider(ctx, in.Provider, in.ProviderRepo)
		if err != nil {
			return err
		}
		repo = remoteRepository.ToRepo(
			parentSpace.ID,
			in.Identifier,
			in.Description,
			&session.Principal,
			c.publicResourceCreationEnabled,
		)

		err = c.repoStore.Create(ctx, repo)
		if err != nil {
			return fmt.Errorf("failed to create repository in storage: %w", err)
		}

		err = c.importer.Run(ctx, provider, repo, remoteRepository.CloneURL, in.Pipelines)
		if err != nil {
			return fmt.Errorf("failed to start import repository job: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	repo.GitURL = c.urlProvider.GenerateGITCloneURL(repo.Path)

	return repo, nil
}

func (c *Controller) sanitizeImportInput(in *ImportInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	if err := c.validateParentRef(in.ParentRef); err != nil {
		return err
	}

	if err := check.RepoIdentifier(in.Identifier); err != nil {
		return err
	}

	if in.Pipelines == "" {
		in.Pipelines = importer.PipelineOptionConvert
	}

	return nil
}
