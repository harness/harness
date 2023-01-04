// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func generateTriggerIDFromEventID(eventID string) string {
	return fmt.Sprintf("event-%s", eventID)
}

func triggerForEventWithGitUID(ctx context.Context, server *Server, repoStore store.RepoStore, eventID string,
	repoGitUID string, triggerType enum.WebhookTrigger, createBody func(*types.Repository) interface{}) error {
	// TODO: can we avoid this DB call? would need the gitrpc to know the repo id though ...
	repo, err := repoStore.FindByGitUID(ctx, repoGitUID)

	// not found error is unrecoverable - most likely a racing condition of repo  being deleted by now
	if err != nil && errors.Is(err, store.ErrResourceNotFound) {
		log.Ctx(ctx).Warn().Err(err).
			Msgf("discard event since repo with gitUID '%s' doesn't exist anymore", repoGitUID)
		return nil
	}

	// all other errors we return and force the event to be reprocessed
	if err != nil {
		return fmt.Errorf("failed to get repo for gitUID '%s': %w", repoGitUID, err)
	}

	body := createBody(repo)
	return triggerForEvent(ctx, server, eventID, enum.WebhookParentRepo, repo.ID, triggerType, body)
}

func triggerForEvent(ctx context.Context, server *Server, eventID string,
	parentType enum.WebhookParent, parentID int64, triggerType enum.WebhookTrigger, body interface{}) error {
	triggerID := generateTriggerIDFromEventID(eventID)

	results, err := server.triggerWebhooksFor(ctx, parentType, parentID, triggerID, triggerType, body)

	// return all errors and force the event to be reprocessed (it's not webhook execution specific!)
	if err != nil {
		return fmt.Errorf("failed to trigger %s (id: '%s') for webhooks of %s %d: %w",
			triggerType, triggerID, parentType, parentID, err)
	}

	// go through all events and figure out if we need to retry the event.
	// Combine all errors into a single error to log (to reduce number of logs)
	retryRequired := false
	var errs error
	for _, result := range results {
		if result.Skipped() {
			continue
		}

		// combine errors of non-successful executions
		if result.Execution.Result != enum.WebhookExecutionResultSuccess {
			errs = multierr.Append(errs, fmt.Errorf("execution %d of webhook %d resulted in %s: %w",
				result.Execution.ID, result.Webhook.ID, result.Execution.Result, result.Err))
		}

		if result.Execution.Result == enum.WebhookExecutionResultRetriableError {
			retryRequired = true
		}
	}

	// in case there was at least one error, log error details in single log to reduce log flooding
	if errs != nil {
		log.Ctx(ctx).Warn().Err(errs).Msgf("webhook execution for %s %d had errors", parentType, parentID)
	}

	// in case at least one webhook has to be retried, return an error to the event framework to have it reprocessed
	if retryRequired {
		return fmt.Errorf("at least one webhook execution resulted in a retry for %s %d", parentType, parentID)
	}

	return nil
}
