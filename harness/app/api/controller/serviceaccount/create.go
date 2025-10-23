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

package serviceaccount

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/dchest/uniuri"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

var (
	serviceAccountUIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	serviceAccountUIDLength   = 16
)

type CreateInput struct {
	Email       string                  `json:"email"`
	DisplayName string                  `json:"display_name"`
	ParentType  enum.ParentResourceType `json:"parent_type"`
	ParentID    int64                   `json:"parent_id"`
}

// Create creates a new service account.
func (c *Controller) Create(ctx context.Context, session *auth.Session,
	in *CreateInput) (*types.ServiceAccount, error) {
	// Ensure principal has required permissions on parent (ensures that parent exists)
	// since it's a create, we use don't pass a resource name.
	if err := apiauth.CheckServiceAccount(ctx, c.authorizer, session, c.spaceStore, c.repoStore,
		in.ParentType, in.ParentID, "", enum.PermissionServiceAccountEdit); err != nil {
		return nil, err
	}

	uid, err := generateServiceAccountUID(in.ParentType, in.ParentID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service account UID: %w", err)
	}

	// TODO: There's a chance of duplicate error - we should retry?
	return c.CreateNoAuth(ctx, in, uid)
}

/*
 * CreateNoAuth creates a new service account without auth checks.
 * WARNING: Never call as part of user flow.
 *
 * Note: take uid separately to allow internally created non-random uids.
 */
func (c *Controller) CreateNoAuth(ctx context.Context,
	in *CreateInput, uid string) (*types.ServiceAccount, error) {
	if err := c.sanitizeCreateInput(in, uid); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	sa := &types.ServiceAccount{
		UID:         uid,
		Email:       in.Email,
		DisplayName: in.DisplayName,
		Salt:        uniuri.NewLen(uniuri.UUIDLen),
		Created:     time.Now().UnixMilli(),
		Updated:     time.Now().UnixMilli(),

		ParentType: in.ParentType,
		ParentID:   in.ParentID,
	}

	err := c.principalStore.CreateServiceAccount(ctx, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput, uid string) error {
	if err := c.principalUIDCheck(uid); err != nil {
		return err
	}

	in.Email = strings.TrimSpace(in.Email)
	if err := check.Email(in.Email); err != nil {
		return err
	}

	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if err := check.DisplayName(in.DisplayName); err != nil {
		return err
	}

	if err := check.ServiceAccountParent(in.ParentType, in.ParentID); err != nil { //nolint:revive
		return err
	}

	return nil
}

// generateServiceAccountUID generates a new unique UID for a service account
// NOTE:
// This method generates 36^10 = ~8*10^24 unique UIDs per parent.
// This should be enough for a very low chance of duplications.
//
// NOTE:
// We generate it automatically to ensure unique UIDs on principals.
// The downside is that they don't have very userfriendly handlers - though that should be okay for service accounts.
// The other option would be take it as an input, but a globally unique uid of a service account
// which itself is scoped to a space / repo might be weird.
func generateServiceAccountUID(parentType enum.ParentResourceType, parentID int64) (string, error) {
	nid, err := gonanoid.Generate(serviceAccountUIDAlphabet, serviceAccountUIDLength)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("sa-%s-%d-%s", string(parentType), parentID, nid), nil
}
