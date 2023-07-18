// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"errors"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// Logout searches for the user's token present in the request and proceeds to  delete it,
// returns nil if successful.
func (c *Controller) Logout(ctx context.Context, session *auth.Session) error {
	var (
		tokenID   int64
		tokenType enum.TokenType
	)

	if session == nil {
		return usererror.BadRequest("no authenticated user")
	}
	switch t := session.Metadata.(type) {
	case *auth.TokenMetadata:
		tokenID = t.TokenID
		tokenType = t.TokenType
	default:
		return errors.New("session metadata is of unknown type")
	}

	if tokenType != enum.TokenTypeSession {
		return usererror.BadRequestf("unsupported logout token type %v", tokenType)
	}

	return c.tokenStore.Delete(ctx, tokenID)
}
