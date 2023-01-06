// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package githook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/auth"
)

// UpdateInput represents the input of the update git hook.
type UpdateInput struct {
	BaseInput
	// RefUpdate contains information about the reference that is being updated.
	RefUpdate ReferenceUpdate `json:"ref_update"`
}

// Update executes the update hook for a git repository.
func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	in *UpdateInput,
) (*ServerHookOutput, error) {
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	// We currently don't have any update action (nothing planned as of now)

	return &ServerHookOutput{}, nil
}
