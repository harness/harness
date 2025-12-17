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
	"slices"
	"strings"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type LinkedSyncInput struct {
	Branches []string `json:"branches"`
}

func (in *LinkedSyncInput) sanitize() error {
	if len(in.Branches) == 0 {
		return errors.InvalidArgument("Branches cannot be empty.")
	}

	for i := range in.Branches {
		in.Branches[i] = strings.TrimSpace(in.Branches[i])

		if in.Branches[i] == "" {
			return errors.InvalidArgument("Branch name cannot be empty.")
		}

		if strings.ContainsAny(in.Branches[i], " :*\t\n\r") {
			return errors.InvalidArgumentf("Invalid branch name %q.", in.Branches[i])
		}
	}

	slices.Sort(in.Branches)
	in.Branches = slices.Compact(in.Branches)

	return nil
}

type LinkedSyncOutput struct {
	Refs []hook.ReferenceUpdate `json:"branches"`
}

func (c *Controller) LinkedSync(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *LinkedSyncInput,
) (*LinkedSyncOutput, error) {
	if err := in.sanitize(); err != nil {
		return nil, err
	}

	repo, err := c.getRepoCheckAccessWithLinked(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, err
	}

	if repo.Type != enum.RepoTypeLinked {
		return nil, errors.InvalidArgument("Repository is not a linked repository.")
	}

	refs := make([]string, len(in.Branches))
	for i := range in.Branches {
		refs[i] = api.BranchPrefix + in.Branches[i]
	}

	linkedRepo, err := c.linkedRepoStore.Find(ctx, repo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find linked repository: %w", err)
	}

	connector := importer.ConnectorDef{
		Path:       linkedRepo.ConnectorPath,
		Identifier: linkedRepo.ConnectorIdentifier,
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc internal write params: %w", err)
	}

	accessInfo, err := c.connectorService.GetAccessInfo(ctx, connector)
	if err != nil {
		return nil, fmt.Errorf("failed to get access info: %w", err)
	}

	cloneURLWithAuth, err := accessInfo.URLWithCredentials()
	if err != nil {
		return nil, errors.InvalidArgument("Failed to get access to repository.")
	}

	result, err := c.git.SyncRefs(ctx, &git.SyncRefsParams{
		WriteParams: writeParams,
		Source:      cloneURLWithAuth,
		Refs:        refs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to synchronize branches: %w", err)
	}

	defaultRef := api.BranchPrefix + repo.DefaultBranch
	updatedDefault := slices.ContainsFunc(result.Refs, func(u hook.ReferenceUpdate) bool {
		return defaultRef == u.Ref
	})

	if updatedDefault {
		repoFull, err := c.repoStore.Find(ctx, repo.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to find repository: %w", err)
		}

		err = c.indexer.Index(ctx, repoFull)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to index repository")
		}
	}

	return &LinkedSyncOutput{Refs: result.Refs}, nil
}
