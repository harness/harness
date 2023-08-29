// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package template

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var (
	// errTemplateRequiresParent if the user tries to create a template without a parent space.
	errTemplateRequiresParent = usererror.BadRequest(
		"Parent space required - standalone templates are not supported.")
)

type CreateInput struct {
	Description string `json:"description"`
	SpaceRef    string `json:"space_ref"` // Ref of the parent space
	UID         string `json:"uid"`
	Type        string `json:"type"`
	Data        string `json:"data"`
}

func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Template, error) {
	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	parentSpace, err := c.spaceStore.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}

	err = apiauth.CheckTemplate(ctx, c.authorizer, session, parentSpace.Path, in.UID, enum.PermissionTemplateEdit)
	if err != nil {
		return nil, err
	}

	var template *types.Template
	now := time.Now().UnixMilli()
	template = &types.Template{
		Description: in.Description,
		Data:        in.Data,
		SpaceID:     parentSpace.ID,
		UID:         in.UID,
		Created:     now,
		Updated:     now,
		Version:     0,
	}
	err = c.templateStore.Create(ctx, template)
	if err != nil {
		return nil, fmt.Errorf("template creation failed: %w", err)
	}

	return template, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)

	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return errTemplateRequiresParent
	}

	if err := c.uidCheck(in.UID, false); err != nil {
		return err
	}

	in.Description = strings.TrimSpace(in.Description)
	return check.Description(in.Description)
}
