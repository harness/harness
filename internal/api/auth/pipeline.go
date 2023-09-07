// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package auth

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
)

// CheckPipeline checks if a pipeline specific permission is granted for the current auth session
// in the scope of the parent.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func CheckPipeline(ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	repoPath string, pipelineUID string, permission enum.Permission) error {
	spacePath, repoName, err := paths.DisectLeaf(repoPath)
	if err != nil {
		return errors.Wrapf(err, "Failed to disect path '%s'", repoPath)
	}
	scope := &types.Scope{SpacePath: spacePath, Repo: repoName}
	resource := &types.Resource{
		Type: enum.ResourceTypePipeline,
		Name: pipelineUID,
	}
	return Check(ctx, authorizer, session, scope, resource, permission)
}
