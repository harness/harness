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
	"fmt"
	"time"

	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const webhookScopeRepo = int64(0)

func (s *Service) sanitizeCreateInput(in *types.WebhookCreateInput, internal bool) error {
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
	if err := CheckURL(in.URL, s.config.AllowLoopback, s.config.AllowPrivateNetwork, internal); err != nil {
		return err
	}
	if err := CheckSecret(in.Secret); err != nil {
		return err
	}
	if err := CheckTriggers(in.Triggers); err != nil { //nolint:revive
		return err
	}

	return nil
}

func (s *Service) Create(
	ctx context.Context,
	principalID int64,
	parentID int64,
	parentType enum.WebhookParent,
	internal bool,
	in *types.WebhookCreateInput,
) (*types.Webhook, error) {
	err := s.sanitizeCreateInput(in, internal)
	if err != nil {
		return nil, err
	}

	encryptedSecret, err := s.encrypter.Encrypt(in.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt webhook secret: %w", err)
	}

	scope := webhookScopeRepo
	if parentType == enum.WebhookParentSpace {
		scope, err = s.spaceStore.GetTreeLevel(ctx, parentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent tree level: %w", err)
		}
	}
	now := time.Now().UnixMilli()

	// create new webhook object
	hook := &types.Webhook{
		ID:         0, // the ID will be populated in the data layer
		Version:    0, // the Version will be populated in the data layer
		CreatedBy:  principalID,
		Created:    now,
		Updated:    now,
		ParentID:   parentID,
		ParentType: parentType,
		Internal:   internal,
		Scope:      scope,

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

	err = s.webhookStore.Create(ctx, hook)

	// internal hooks are hidden from non-internal read requests - properly communicate their existence on duplicate.
	// This is best effort, any error we just ignore and fallback to original duplicate error.
	if errors.Is(err, store.ErrDuplicate) && !internal {
		existingHook, derr := s.webhookStore.FindByIdentifier(
			ctx, enum.WebhookParentRepo, parentID, hook.Identifier)
		if derr != nil {
			log.Ctx(ctx).Warn().Err(derr).Msgf(
				"failed to retrieve webhook for repo %d with identifier %q on duplicate error",
				parentID,
				hook.Identifier,
			)
		}
		if derr == nil && existingHook.Internal {
			return nil, errors.Conflict("The provided identifier is reserved for internal purposes.")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to store webhook: %w", err)
	}

	return hook, nil
}
