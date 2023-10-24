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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
)

// CheckRepo checks if a repo specific permission is granted for the current auth session
// in the scope of its parent.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func CheckRepo(
	ctx context.Context,
	authorizer authz.Authorizer,
	session *auth.Session,
	repo *types.Repository,
	permission enum.Permission,
	orPublic bool,
) error {
	if orPublic && repo.IsPublic {
		return nil
	}

	parentSpace, name, err := paths.DisectLeaf(repo.Path)
	if err != nil {
		return errors.Wrapf(err, "Failed to disect path '%s'", repo.Path)
	}

	scope := &types.Scope{SpacePath: parentSpace}
	resource := &types.Resource{
		Type: enum.ResourceTypeRepo,
		Name: name,
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}

func IsSpaceAdmin(
	ctx context.Context,
	authorizer authz.Authorizer,
	session *auth.Session,
	repo *types.Repository,
) (bool, error) {
	err := CheckRepo(ctx, authorizer, session, repo, enum.PermissionSpaceCreate, false)
	if err != nil && !errors.Is(err, ErrNotAuthorized) {
		return false, fmt.Errorf("failed to check access to find if the user is space admin: %w", err)
	}

	return err == nil, nil
}
