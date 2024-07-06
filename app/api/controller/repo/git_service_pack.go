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
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types/enum"
)

// GitServicePack executes the service pack part of git's smart http protocol (receive-/upload-pack).
func (c *Controller) GitServicePack(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	options api.ServicePackOptions,
) error {
	isWriteOperation := false
	permission := enum.PermissionRepoView
	// receive-pack is the server receiving data - aka the client pushing data.
	if options.Service == enum.GitServiceTypeReceivePack {
		isWriteOperation = true
		permission = enum.PermissionRepoPush
	}

	repo, err := c.getRepoCheckAccessForGit(ctx, session, repoRef, permission)
	if err != nil {
		return fmt.Errorf("failed to verify repo access: %w", err)
	}

	params := &git.ServicePackParams{
		// TODO: git shouldn't take a random string here, but instead have accepted enum values.
		ServicePackOptions: options,
	}

	// setup read/writeparams depending on whether it's a write operation
	if isWriteOperation {
		var writeParams git.WriteParams
		writeParams, err = controller.CreateRPCExternalWriteParams(ctx, c.urlProvider, session, repo)
		if err != nil {
			return fmt.Errorf("failed to create RPC write params: %w", err)
		}
		params.WriteParams = &writeParams
	} else {
		readParams := git.CreateReadParams(repo)
		params.ReadParams = &readParams
	}

	if err = c.git.ServicePack(ctx, params); err != nil {
		return fmt.Errorf("failed service pack operation %q  on git: %w", options.Service, err)
	}

	return nil
}
