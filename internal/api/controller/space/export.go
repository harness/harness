// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/services/exporter"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type ExportInput struct {
	AccountId         string `json:"accountId"`
	OrgIdentifier     string `json:"orgIdentifier"`
	ProjectIdentifier string `json:"projectIdentifier"`
	Token             string `json:"token"`
}

// Export creates a new empty repository in harness code and does git push to it.
func (c *Controller) Export(ctx context.Context, session *auth.Session, spaceRef string, in *ExportInput) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceEdit, false); err != nil {
		return err
	}

	err = c.sanitizeExportInput(in)
	if err != nil {
		return fmt.Errorf("failed to sanitize input: %w", err)
	}

	providerInfo := &exporter.HarnessCodeInfo{
		AccountId:         in.AccountId,
		ProjectIdentifier: in.ProjectIdentifier,
		OrgIdentifier:     in.OrgIdentifier,
		Token:             in.Token,
	}

	// todo(abhinav): add pagination
	repos, err := c.repoStore.List(ctx, space.ID, &types.RepoFilter{Size: 200})
	if err != nil {
		return err
	}

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		err = c.exporter.RunMany(ctx, space.ID, providerInfo, repos)
		if err != nil {
			return fmt.Errorf("failed to start export repository job: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) sanitizeExportInput(in *ExportInput) error {
	if in.AccountId == "" {
		return usererror.BadRequest("account id must be provided")
	}

	if in.OrgIdentifier == "" {
		return usererror.BadRequest("organization identifier must be provided")
	}

	if in.ProjectIdentifier == "" {
		return usererror.BadRequest("project identifier must be provided")
	}

	if in.Token == "" {
		return usererror.BadRequest("token for harness code must be provided")
	}

	return nil
}
