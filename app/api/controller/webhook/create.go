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

package webhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type CreateInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID        string `json:"uid" deprecated:"true"`
	Identifier string `json:"identifier"`
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	DisplayName string                `json:"display_name"`
	Description string                `json:"description"`
	URL         string                `json:"url"`
	Secret      string                `json:"secret"`
	Enabled     bool                  `json:"enabled"`
	Insecure    bool                  `json:"insecure"`
	Triggers    []enum.WebhookTrigger `json:"triggers"`
}

// Create creates a new webhook.
//
//nolint:gocognit
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateInput,
	internal bool,
) (*types.Webhook, error) {
	// validate input
	err := sanitizeCreateInput(in, c.allowLoopback, c.allowPrivateNetwork || internal)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	encryptedSecret, err := c.encrypter.Encrypt(in.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt webhook secret: %w", err)
	}

	// create new webhook object
	hook := &types.Webhook{
		ID:         0, // the ID will be populated in the data layer
		Version:    0, // the Version will be populated in the data layer
		CreatedBy:  session.Principal.ID,
		Created:    now,
		Updated:    now,
		ParentID:   repo.ID,
		ParentType: enum.WebhookParentRepo,
		Internal:   internal,

		// user input
		Identifier:            in.Identifier,
		DisplayName:           in.DisplayName,
		Description:           in.Description,
		URL:                   in.URL,
		Secret:                string(encryptedSecret),
		Enabled:               in.Enabled,
		Insecure:              in.Insecure,
		Triggers:              DeduplicateTriggers(in.Triggers),
		LatestExecutionResult: nil,
	}

	err = c.webhookStore.Create(ctx, hook)

	// internal hooks are hidden from non-internal read requests - properly communicate their existence on duplicate.
	// This is best effort, any error we just ignore and fallback to original duplicate error.
	if errors.Is(err, store.ErrDuplicate) && !internal {
		existingHook, derr := c.webhookStore.FindByIdentifier(ctx, enum.WebhookParentRepo, repo.ID, hook.Identifier)
		if derr != nil {
			log.Ctx(ctx).Warn().Err(derr).Msgf(
				"failed to retrieve webhook for repo %d with identifier %q on duplicate error",
				repo.ID,
				hook.Identifier,
			)
		}
		if derr == nil && existingHook.Internal {
			return nil, usererror.Conflict("The provided identifier is reserved for internal purposes.")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to store webhook: %w", err)
	}

	return hook, nil
}

func sanitizeCreateInput(in *CreateInput, allowLoopback bool, allowPrivateNetwork bool) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	// backfill required data - during migration period we have to accept both, displayname only and identifier only
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed
	if in.DisplayName == "" && in.Identifier != "" {
		in.DisplayName = in.Identifier
	}
	if in.Identifier == "" && in.DisplayName != "" {
		var err error
		in.Identifier, err = migrate.WebhookDisplayNameToIdentifier(in.DisplayName, false)
		if err != nil {
			return fmt.Errorf("failed to migrate webhook displayname %q to identifier: %w", in.DisplayName, err)
		}
	}

	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}
	if err := check.DisplayName(in.DisplayName); err != nil {
		return err
	}
	if err := check.Description(in.Description); err != nil {
		return err
	}
	if err := CheckURL(in.URL, allowLoopback, allowPrivateNetwork); err != nil {
		return err
	}
	if err := checkSecret(in.Secret); err != nil {
		return err
	}
	if err := CheckTriggers(in.Triggers); err != nil { //nolint:revive
		return err
	}

	return nil
}
