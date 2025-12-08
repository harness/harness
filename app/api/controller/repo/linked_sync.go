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
	"strings"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/types/enum"
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

	return nil
}

func (c *Controller) LinkedSync(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *LinkedSyncInput,
) error {
	if err := in.sanitize(); err != nil {
		return err
	}

	repo, err := c.getRepoCheckAccessWithLinked(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return err
	}

	if repo.Type != enum.RepoTypeLinked {
		return errors.InvalidArgument("Repository is not a linked repository.")
	}

	refSpec := make([]string, len(in.Branches))
	for i := range in.Branches {
		ref, err := git.GetRefPath(in.Branches[i], gitenum.RefTypeBranch)
		if err != nil {
			return fmt.Errorf("failed to get ref path for branch %s: %w", in.Branches[i], err)
		}

		refSpec[i] = ref + ":" + ref
	}

	linkedRepo, err := c.linkedRepoStore.Find(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to find linked repository: %w", err)
	}

	connector := importer.ConnectorDef{
		Path:       linkedRepo.ConnectorPath,
		Identifier: linkedRepo.ConnectorIdentifier,
		Repo:       linkedRepo.ConnectorRepo,
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return fmt.Errorf("failed to create rpc internal write params: %w", err)
	}

	cloneURLWithAuth, err := importer.ConnectorToURL(ctx, c.connectorService, connector)
	if err != nil {
		return errors.InvalidArgument("Failed to get access to repository.")
	}

	_, err = c.git.SyncRepository(ctx, &git.SyncRepositoryParams{
		WriteParams:       writeParams,
		Source:            cloneURLWithAuth,
		CreateIfNotExists: false,
		RefSpecs:          refSpec,
		DefaultBranch:     repo.DefaultBranch,
	})
	if err != nil {
		return fmt.Errorf("failed to synchronize branches: %w", err)
	}

	return nil
}
