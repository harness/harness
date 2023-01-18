// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// CreatePathInput used for path creation apis.
type CreatePathInput struct {
	Path string `json:"path"`
}

// CreatePath creates a new path for a repo.
func (c *Controller) CreatePath(ctx context.Context, session *auth.Session,
	repoRef string, in *CreatePathInput) (*types.Path, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	if err = c.sanitizeCreatePathInput(in, repo.Path); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	now := time.Now().UnixMilli()
	path := &types.Path{
		Version:    0,
		Value:      in.Path,
		IsPrimary:  false,
		TargetType: enum.PathTargetTypeRepo,
		TargetID:   repo.ID,
		CreatedBy:  session.Principal.ID,
		Created:    now,
		Updated:    now,
	}

	// TODO: ensure principal is authorized to create a path pointing to in.Path
	err = c.pathStore.Create(ctx, path)
	if err != nil {
		return nil, err
	}

	return path, nil
}

func (c *Controller) sanitizeCreatePathInput(in *CreatePathInput, oldPath string) error {
	in.Path = strings.Trim(in.Path, "/")

	if err := check.Path(in.Path, false, c.uidCheck); err != nil {
		return err
	}

	rootSeparatorIdx := strings.Index(in.Path, types.PathSeparator)
	if rootSeparatorIdx < 0 {
		return usererror.BadRequest("Top level paths are not allowed for repositories.")
	}

	// add '/' at the end of the root uid to avoid false negatives (e.g. abc/repo-> abcdef/repo)
	newPathRoot := in.Path[:rootSeparatorIdx] + types.PathSeparator
	if !strings.HasPrefix(oldPath, newPathRoot) {
		return usererror.BadRequest("Path has to stay within the same top level space.")
	}

	return nil
}
