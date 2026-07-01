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
	gitgitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RefSyncEntry describes one ref to fetch from the remote.
// If LocalRef is empty the ref is stored verbatim under RemoteRef.
// If LocalRef is non-empty the ref is fetched under RemoteRef then atomically
// renamed to LocalRef, and the temporary RemoteRef is deleted.
type RefSyncEntry struct {
	RemoteRef string
	LocalRef  string
}

// RunSyncRefs resolves the connector's access info, builds the auth'd clone
// URL, and runs git SyncRefs against the local mirror. In OSS gitness mode
// connectorService is a noop and SyncRefs errors.
//
// Entries with an empty LocalRef are stored verbatim. Entries with a non-empty
// LocalRef are fetched under RemoteRef then renamed to LocalRef via a local
// UpdateRefs call (because the gitrpc transport does not carry rename metadata).
func RunSyncRefs(
	ctx context.Context,
	gitClient git.Interface,
	repoFinder refcache.RepoFinder,
	urlProvider gitnessurl.Provider,
	connectorService importer.ConnectorService,
	linkedRepo *types.LinkedRepo,
	entries []RefSyncEntry,
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

	// Fetch all refs by their RemoteRef names. Entries whose LocalRef differs
	// are renamed with a subsequent UpdateRefs call; the gitrpc transport does
	// not carry rename metadata so we handle it locally after the fetch.
	fetchRefs := make([]string, 0, len(entries))
	for _, e := range entries {
		fetchRefs = append(fetchRefs, e.RemoteRef)
	}

	if _, err := gitClient.SyncRefs(ctx, &git.SyncRefsParams{
		WriteParams: writeParams,
		Source:      cloneURL,
		Refs:        fetchRefs,
	}); err != nil {
		return fmt.Errorf("linkedpr: git SyncRefs: %w", err)
	}

	// For each entry that needs renaming: read the fetched SHA, point LocalRef
	// at it, and delete the temporary RemoteRef in one atomic UpdateRefs call.
	var renameUpdates []git.RefUpdate
	for _, e := range entries {
		if e.LocalRef == "" {
			continue // stored verbatim, nothing more to do
		}

		resp, err := gitClient.GetRef(ctx, git.GetRefParams{
			ReadParams: git.ReadParams{RepoUID: repo.GitUID},
			Name:       e.RemoteRef,
			Type:       gitgitenum.RefTypeRaw,
		})
		if err != nil {
			// SyncRefs succeeded with this ref in fetchRefs, so a GetRef failure
			// here indicates internal inconsistency — surface it rather than skipping.
			return fmt.Errorf("linkedpr: GetRef %s after sync: %w", e.RemoteRef, err)
		}

		renameUpdates = append(renameUpdates,
			git.RefUpdate{Name: e.LocalRef, New: resp.SHA},
			git.RefUpdate{Name: e.RemoteRef, New: sha.None},
		)
	}

	if len(renameUpdates) == 0 {
		return nil
	}

	if err := gitClient.UpdateRefs(ctx, git.UpdateRefsParams{
		WriteParams: writeParams,
		Refs:        renameUpdates,
	}); err != nil {
		return fmt.Errorf("linkedpr: rename provider refs to gitness namespace: %w", err)
	}

	return nil
}
