// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

var ()

// Logout searches for the user's token present in the request and proceeds to  delete it.
// If no user was present, a usererror.ErrUnauthorized is returned.
func (c *Controller) Logout(ctx context.Context, session *auth.Session) error {
	var (
		tokenID   int64
		tokenType enum.TokenType
	)

	if session == nil {
		return usererror.ErrUnauthorized
	}

	switch t := session.Metadata.(type) {
	case *auth.TokenMetadata:
		tokenID = t.TokenID
		tokenType = t.TokenType
	default:
		return errors.New("provided jwt doesn't support logout")
	}

	if tokenType != enum.TokenTypeSession {
		return usererror.BadRequestf("unsupported logout token type %v", tokenType)
	}

	err := c.tokenStore.Delete(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to delete token from store: %w", err)
	}

	return nil
}
