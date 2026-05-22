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

package repo

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types/enum"
)

// webhookRegistrationUserError classifies a WebhookService error and returns
// an appropriate user-facing error. The underlying cause is already logged by
// the WebhookService layer; here we just pick a response the user can act on.
func webhookRegistrationUserError(err error) error {
	if stderrors.Is(err, importer.ErrWebhookUpsertUserFault) {
		return usererror.BadRequest(
			"Failed to register webhook for the linked repository. " +
				"Please verify the connector's token has webhook read/write " +
				"permission on the target repository and try again.",
		)
	}
	// Infra / unknown: keep as Internal; user can't fix it, ops needs to look.
	return errors.Internal(err, "Failed to register webhook for the linked repository.")
}

// LinkedRegisterWebhook re-registers the provider-side webhook for an existing
// linked repo. Safe to call repeatedly: SCM-service upserts by URL, so an
// existing hook is updated in place and a missing one is recreated.
//
// Returns no body on success; on failure returns the same HTTP-status shape as
// the create endpoint (400 for user/token issues, 500 for infra issues).
func (c *Controller) LinkedRegisterWebhook(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
) error {
	repo, err := c.getRepoCheckAccessWithLinked(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return err
	}

	if repo.Type != enum.RepoTypeLinked {
		return errors.InvalidArgument("Repository is not a linked repository.")
	}

	linkedRepo, err := c.linkedRepoStore.Find(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to find linked repository: %w", err)
	}

	parentSpacePath := paths.Parent(repo.Path)

	accessInfo, err := c.connectorService.GetAccessInfo(ctx, importer.ConnectorDef{
		Path:           linkedRepo.ConnectorPath,
		Identifier:     linkedRepo.ConnectorIdentifier,
		RepoIdentifier: linkedRepo.ConnectorRepo,
	})
	if err != nil {
		return fmt.Errorf("failed to get connector access info: %w", err)
	}
	if accessInfo.URL == "" {
		return errors.Internal(nil, "connector returned an empty repository URL")
	}

	if err := c.webhookService.UpsertWebhook(ctx, importer.UpsertWebhookInput{
		SpacePath:           parentSpacePath,
		ConnectorPath:       linkedRepo.ConnectorPath,
		ConnectorIdentifier: linkedRepo.ConnectorIdentifier,
		CloneURL:            accessInfo.URL,
	}); err != nil {
		return webhookRegistrationUserError(err)
	}

	return nil
}
