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

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func generateTriggerIDFromEventID(eventID string) string {
	return fmt.Sprintf("event-%s", eventID)
}

// triggerForEventWithRepo triggers all webhooks for the given repo and triggerType
// using the eventID to generate a deterministic triggerID and using the output of bodyFn as payload.
// The method tries to find the repository and principal and provides both to the bodyFn to generate the body.
// NOTE: technically we could avoid this call if we send the data via the event (though then events will get big).
func (s *Service) triggerForEventWithRepo(
	ctx context.Context,
	triggerType enum.WebhookTrigger,
	eventID string,
	principalID int64,
	repoID int64,
	createBodyFn func(*types.Principal, *types.Repository) (any, error),
) error {
	principal, err := s.findPrincipalForEvent(ctx, principalID)
	if err != nil {
		return err
	}

	repo, err := s.findRepositoryForEvent(ctx, repoID)
	if err != nil {
		return err
	}

	// create body
	body, err := createBodyFn(principal, repo)
	if err != nil {
		return fmt.Errorf("body creation function failed: %w", err)
	}

	parents, err := s.getParentInfoRepo(ctx, repo.ID, true)
	if err != nil {
		return fmt.Errorf("failed to get webhook parent info for parents: %w", err)
	}

	return s.triggerForEvent(ctx, eventID, parents, triggerType, body)
}

// triggerForEventWithPullReq triggers all webhooks for the given repo and triggerType
// using the eventID to generate a deterministic triggerID and using the output of bodyFn as payload.
// The method tries to find the pullreq, principal, target repo, and source repo
// and provides all to the bodyFn to generate the body.
// NOTE: technically we could avoid this call if we send the data via the event (though then events will get big).
func (s *Service) triggerForEventWithPullReq(ctx context.Context,
	triggerType enum.WebhookTrigger, eventID string, principalID int64, prID int64,
	createBodyFn func(principal *types.Principal, pr *types.PullReq,
		targetRepo *types.Repository, sourceRepo *types.Repository) (any, error)) error {
	principal, err := s.findPrincipalForEvent(ctx, principalID)
	if err != nil {
		return err
	}

	pr, err := s.findPullReqForEvent(ctx, prID)
	if err != nil {
		return err
	}

	targetRepo, err := s.findRepositoryForEvent(ctx, pr.TargetRepoID)
	if err != nil {
		return fmt.Errorf("failed to get pr target repo: %w", err)
	}

	sourceRepo := targetRepo
	if pr.SourceRepoID != pr.TargetRepoID {
		sourceRepo, err = s.findRepositoryForEvent(ctx, pr.SourceRepoID)
		if err != nil {
			return fmt.Errorf("failed to get pr source repo: %w", err)
		}
	}

	// create body
	body, err := createBodyFn(principal, pr, targetRepo, sourceRepo)
	if err != nil {
		return fmt.Errorf("body creation function failed: %w", err)
	}

	parents, err := s.getParentInfoRepo(ctx, targetRepo.ID, true)
	if err != nil {
		return fmt.Errorf("failed to get webhook parent info: %w", err)
	}

	return s.triggerForEvent(ctx, eventID, parents, triggerType, body)
}

// findRepositoryForEvent finds the repository for the provided repoID.
func (s *Service) findRepositoryForEvent(ctx context.Context, repoID int64) (*types.Repository, error) {
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

// findPullReqForEvent finds the pullrequest for the provided prID.
func (s *Service) findPullReqForEvent(ctx context.Context, prID int64) (*types.PullReq, error) {
	pr, err := s.pullreqStore.Find(ctx, prID)

	if err != nil && errors.Is(err, store.ErrResourceNotFound) {
		// not found error is unrecoverable - most likely a racing condition of repo being deleted by now
		return nil, events.NewDiscardEventErrorf("PR with id '%d' doesn't exist anymore", prID)
	}
	if err != nil {
		// all other errors we return and force the event to be reprocessed
		return nil, fmt.Errorf("failed to get PR for id '%d': %w", prID, err)
	}

	return pr, nil
}

// findPrincipalForEvent finds the principal for the provided principalID.
func (s *Service) findPrincipalForEvent(ctx context.Context, principalID int64) (*types.Principal, error) {
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

// triggerForEvent triggers all webhooks for the given parentType/ID and triggerType
// using the eventID to generate a deterministic triggerID and sending the provided body as payload.
func (s *Service) triggerForEvent(
	ctx context.Context,
	eventID string,
	parents []types.WebhookParentInfo,
	triggerType enum.WebhookTrigger,
	body any,
) error {
	triggerID := generateTriggerIDFromEventID(eventID)

	results, err := s.triggerWebhooksFor(ctx, parents, triggerID, triggerType, body)

	// return all errors and force the event to be reprocessed (it's not webhook execution specific!)
	if err != nil {
		return fmt.Errorf(
			"failed to trigger %s (id: '%s') for webhooks %#v: %w",
			triggerType, triggerID, parents, err,
		)
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
			errs = multierr.Append(errs,
				fmt.Errorf("execution %d of webhook %d resulted in %s: %w",
					result.Execution.ID, result.Webhook.ID, result.Execution.Result, result.Err))
		}

		if result.Execution.Result == enum.WebhookExecutionResultRetriableError {
			retryRequired = true
		}
	}

	// in case there was at least one error, log error details in single log to reduce log flooding
	if errs != nil {
		log.Ctx(ctx).Warn().Err(errs).Msgf("webhook execution for %#v had errors", parents)
	}

	// in case at least one webhook has to be retried, return an error to the event framework to have it reprocessed
	if retryRequired {
		return fmt.Errorf("at least one webhook execution resulted in a retry for %#v", parents)
	}

	return nil
}
