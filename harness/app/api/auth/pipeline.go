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
)

// CheckPipeline checks if a pipeline specific permission is granted for the current auth session
// in the scope of the parent.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func CheckPipeline(ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	repoPath string, pipelineIdentifier string, permission enum.Permission) error {
	spacePath, repoName, err := paths.DisectLeaf(repoPath)
	if err != nil {
		return fmt.Errorf("failed to disect path '%s': %w", repoPath, err)
	}
	scope := &types.Scope{SpacePath: spacePath, Repo: repoName}
	resource := &types.Resource{
		Type:       enum.ResourceTypePipeline,
		Identifier: pipelineIdentifier,
	}
	return Check(ctx, authorizer, session, scope, resource, permission)
}
