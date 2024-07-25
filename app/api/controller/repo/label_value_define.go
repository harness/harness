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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// DefineLabelValue defines a new label value for the specified repository.
func (c *Controller) DefineLabelValue(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	key string,
	in *types.DefineValueInput,
) (*types.LabelValue, error) {
	repo, err := GetRepo(ctx, c.repoStore, repoRef, []enum.RepoState{enum.RepoStateActive})
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate input: %w", err)
	}

	label, err := c.labelSvc.Find(ctx, nil, &repo.ID, key)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo label: %w", err)
	}

	permission := enum.PermissionRepoEdit
	if label.Type == enum.LabelTypeDynamic {
		permission = enum.PermissionRepoPush
	}

	if err = apiauth.CheckRepo(
		ctx, c.authorizer, session, repo, permission); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	value, err := c.labelSvc.DefineValue(ctx, session.Principal.ID, label.ID, in)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo label value: %w", err)
	}

	return value, nil
}
