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

package handlers

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/refcache"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RunSyncRefs resolves the connector's access info, builds the auth'd clone
// URL, and runs git SyncRefs against the local mirror. In OSS gitness mode
// connectorService is a noop and SyncRefs errors.
func RunSyncRefs(
	ctx context.Context,
	gitClient git.Interface,
	repoFinder refcache.RepoFinder,
	urlProvider gitnessurl.Provider,
	connectorService importer.ConnectorService,
	linkedRepo *types.LinkedRepo,
	refs []string,
) error {
	accessInfo, err := connectorService.GetAccessInfo(ctx, importer.ConnectorDef{
		Path:           linkedRepo.ConnectorPath,
		Identifier:     linkedRepo.ConnectorIdentifier,
		RepoIdentifier: linkedRepo.ConnectorRepo,
	})
	if err != nil {
		return fmt.Errorf("linkedpr: get connector access info: %w", err)
	}

	cloneURL, err := accessInfo.URLWithCredentials()
	if err != nil {
		return fmt.Errorf("linkedpr: build clone url with creds: %w", err)
	}

	repo, err := repoFinder.FindByID(ctx, linkedRepo.RepoID)
	if err != nil {
		return fmt.Errorf("linkedpr: find repo %d: %w", linkedRepo.RepoID, err)
	}

	systemPrincipal := bootstrap.NewSystemServiceSession().Principal
	envVars, err := githook.GenerateEnvironmentVariablesForOperation(
		ctx,
		urlProvider.GetInternalAPIURL(ctx),
		repo.ID,
		systemPrincipal.ID,
		true,
		enum.GitOpTypeManageRepo,
	)
	if err != nil {
		return fmt.Errorf("linkedpr: githook env vars: %w", err)
	}

	writeParams := git.WriteParams{
		RepoUID: repo.GitUID,
		Actor: git.Identity{
			Name:  systemPrincipal.DisplayName,
			Email: systemPrincipal.Email,
		},
		EnvVars: envVars,
	}

	if _, err := gitClient.SyncRefs(ctx, &git.SyncRefsParams{
		WriteParams: writeParams,
		Source:      cloneURL,
		Refs:        refs,
	}); err != nil {
		return fmt.Errorf("linkedpr: git SyncRefs: %w", err)
	}
	return nil
}
