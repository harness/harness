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

package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
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
		return usererror.BadRequestf("Unsupported logout token type %v", tokenType)
	}

	err := c.tokenStore.Delete(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to delete token from store: %w", err)
	}

	return nil
}
