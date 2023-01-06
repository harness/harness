// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func generateTriggerIDFromEventID(eventID string) string {
	return fmt.Sprintf("event-%s", eventID)
}

// triggerWebhooksForEventWithRepoAndPrincipal triggers all webhooks for the given repo and triggerType
// using the eventID to generate a deterministic triggerID and using the output of bodyFn as payload.
// The method tries to find the repository and principal and provides both to the bodyFn to generate the body.
func (s *Server) triggerWebhooksForEventWithRepoAndPrincipal(ctx context.Context,
	triggerType enum.WebhookTrigger, eventID string, repoID int64, principalID int64,
	createBodyFn func(*types.Repository, *types.Principal) interface{}) error {
	// NOTE: technically we could avoid this call if we send the data via the event (though then events will get big)
	repo, err := s.findRepositoryForEvent(ctx, repoID)
	if err != nil {
		return err
	}

	// NOTE: technically we could avoid this call if we send the data via the event (though then events will get big)
	principal, err := s.findPrincipalForEvent(ctx, principalID)
	if err != nil {
		return err
	}

	// create body
	body := createBodyFn(repo, principal)

	return s.triggerWebhooksForEvent(ctx, eventID, enum.WebhookParentRepo, repo.ID, triggerType, body)
}

// findRepositoryForEvent finds the repository for the provided repoID.
func (s *Server) findRepositoryForEvent(ctx context.Context, repoID int64) (*types.Repository, error) {
	repo, err := s.repoStore.Find(ctx, repoID)

	if err != nil && errors.Is(err, store.ErrResourceNotFound) {
		// not found error is unrecoverable - most likely a racing condition of repo being deleted by now
		return nil, events.NewDiscardEventErrorf("repo with id '%d' doesn't exist anymore", repoID)
	}
	if err != nil {
		// all other errors we return and force the event to be reprocessed
		return nil, fmt.Errorf("failed to get repo for id '%d': %w", repoID, err)
	}

	return repo, nil
}

// findPrincipalForEvent finds the principal for the provided principalID.
func (s *Server) findPrincipalForEvent(ctx context.Context, principalID int64) (*types.Principal, error) {
	principal, err := s.principalStore.Find(ctx, principalID)

	if err != nil && errors.Is(err, store.ErrResourceNotFound) {
		// this should never happen (as we won't delete principals) - discard event
		return nil, events.NewDiscardEventErrorf("principal with id '%d' doesn't exist anymore", principalID)
	}
	if err != nil {
		// all other errors we return and force the event to be reprocessed
		return nil, fmt.Errorf("failed to get principal for id '%d': %w", principalID, err)
	}

	return principal, nil
}

// triggerWebhooksForEvent triggers all webhooks for the given parentType/ID and triggerType
// using the eventID to generate a deterministic triggerID and sending the provided body as payload.
func (s *Server) triggerWebhooksForEvent(ctx context.Context, eventID string,
	parentType enum.WebhookParent, parentID int64, triggerType enum.WebhookTrigger, body interface{}) error {
	triggerID := generateTriggerIDFromEventID(eventID)

	results, err := s.triggerWebhooksFor(ctx, parentType, parentID, triggerID, triggerType, body)

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
