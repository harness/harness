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

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types/enum"
)

// DeleteTag deletes a tag from the repo.
func (c *Controller) DeleteTag(ctx context.Context,
	session *auth.Session,
	repoRef,
	tagName string,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush, false)
	if err != nil {
		return err
	}

	// TODO: Verify protection rules

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return fmt.Errorf("failed to create RPC write params: %w", err)
	}

	err = c.gitRPCClient.DeleteTag(ctx, &gitrpc.DeleteTagParams{
		Name:        tagName,
		WriteParams: writeParams,
	})
	if err != nil {
		return err
	}
	return nil
}
