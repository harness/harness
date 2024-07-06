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
	"io"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types/enum"
)

// GitInfoRefs executes the info refs part of git's smart http protocol.
func (c *Controller) GitInfoRefs(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	service enum.GitServiceType,
	gitProtocol string,
	w io.Writer,
) error {
	repo, err := c.getRepoCheckAccessForGit(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return fmt.Errorf("failed to verify repo access: %w", err)
	}

	if err = c.git.GetInfoRefs(ctx, w, &git.InfoRefsParams{
		ReadParams: git.CreateReadParams(repo),
		// TODO: git shouldn't take a random string here, but instead have accepted enum values.
		Service:     string(service),
		Options:     nil,
		GitProtocol: gitProtocol,
	}); err != nil {
		return fmt.Errorf("failed GetInfoRefs on git: %w", err)
	}

	return nil
}
