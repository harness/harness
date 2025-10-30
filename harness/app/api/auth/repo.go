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

package auth

import (
	"context"
	"fmt"
	"slices"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CheckRepo checks if a repo specific permission is granted for the current auth session
// in the scope of its parent.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func CheckRepo(
	ctx context.Context,
	authorizer authz.Authorizer,
	session *auth.Session,
	repo *types.RepositoryCore,
	permission enum.Permission,
) error {
	parentSpace, name, err := paths.DisectLeaf(repo.Path)
	if err != nil {
		return fmt.Errorf("failed to disect path '%s': %w", repo.Path, err)
	}

	scope := &types.Scope{SpacePath: parentSpace}
	resource := &types.Resource{
		Type:       enum.ResourceTypeRepo,
		Identifier: name,
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}

func IsRepoOwner(
	ctx context.Context,
	authorizer authz.Authorizer,
	session *auth.Session,
	repo *types.RepositoryCore,
) (bool, error) {
	// for now we use repoedit as permission to verify if someone is a SpaceOwner and hence a RepoOwner.
	err := CheckRepo(ctx, authorizer, session, repo, enum.PermissionRepoEdit)
	if err != nil && !IsNoAccess(err) {
		return false, fmt.Errorf("failed to check access user access: %w", err)
	}

	return err == nil, nil
}

// CheckRepoState checks if requested permission is allowed given the state of the repository.
func CheckRepoState(
	_ context.Context,
	_ *auth.Session,
	repo *types.RepositoryCore,
	reqPermission enum.Permission,
	additionalAllowedRepoStates ...enum.RepoState,
) error {
	permissionsAllowedPerRepoState := map[enum.RepoState][]enum.Permission{
		enum.RepoStateActive: {
			enum.PermissionRepoView,
			enum.PermissionRepoCreate,
			enum.PermissionRepoEdit,
			enum.PermissionRepoPush,
			enum.PermissionRepoReview,
			enum.PermissionRepoDelete,
			enum.PermissionRepoReportCommitCheck,

			enum.PermissionPipelineView,
			enum.PermissionPipelineExecute,
			enum.PermissionPipelineEdit,
			enum.PermissionPipelineDelete,

			enum.PermissionServiceAccountView,
		},

		enum.RepoStateArchived: {
			enum.PermissionRepoView,

			enum.PermissionPipelineView,

			enum.PermissionServiceAccountView,
		},

		// allowed permissions for repos on transition states during import/migration are handled by their controller.
		enum.RepoStateGitImport:         {},
		enum.RepoStateMigrateDataImport: {},
		enum.RepoStateMigrateGitPush:    {},
	}

	if len(additionalAllowedRepoStates) > 0 && slices.Contains(additionalAllowedRepoStates, repo.State) {
		return nil
	}

	defaultAllowedPermissions := permissionsAllowedPerRepoState[repo.State]
	if !slices.Contains(defaultAllowedPermissions, reqPermission) {
		return errors.PreconditionFailed("Operation is not allowed for repository in state %s", repo.State)
	}

	return nil
}
