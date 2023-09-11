// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/services/exporter"
	"github.com/harness/gitness/types"
	"github.com/rs/zerolog/log"
)

type ExportInput struct {
	ParentRef string `json:"parent_ref"`

	AccountId         string `json:"accountId"`
	OrgIdentifier     string `json:"orgIdentifier"`
	ProjectIdentifier string `json:"projectIdentifier"`
	Token             string `json:"token"`
}

// Export creates a new empty repository in harness code and does git push to it.
func (c *Controller) Export(ctx context.Context, session *auth.Session, in *ExportInput) (*types.Repository, error) {
	// todo(abhinav): check perms
	parentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	err = c.sanitizeExportInput(in)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	providerInfo := exporter.HarnessCodeInfo{
		AccountId:         in.AccountId,
		ProjectIdentifier: in.ProjectIdentifier,
		OrgIdentifier:     in.OrgIdentifier,
		Token:             in.Token,
	}

	// todo(abhinav): add pagination
	repos, err := c.repoStore.List(ctx, parentSpace.ID, &types.RepoFilter{})
	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Info().Msgf("Add migration for repos ", providerInfo, repos)

	return nil, nil
}

func (c *Controller) sanitizeExportInput(in *ExportInput) error {
	if err := c.validateParentRef(in.ParentRef); err != nil {
		return err
	}

	if in.AccountId == "" {
		return usererror.BadRequest("account id must be provided")
	}

	if in.OrgIdentifier == "" {
		return usererror.BadRequest("organization identifier must be provided")
	}

	if in.ProjectIdentifier == "" {
		return usererror.BadRequest("project identifier must be provided")
	}

	return nil
}
