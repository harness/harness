// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/gitrpc/events"

	"github.com/rs/zerolog/log"
)

func branchCreated(ctx context.Context, event *events.Event[*gitevents.BranchCreatedPayload]) error {
	log.Ctx(ctx).Info().Msgf("branch '%s' (SHA: '%s') got created in repo '%s' at %s",
		event.Payload.BranchName, event.Payload.SHA, event.Payload.RepoUID, event.Timestamp)
	return nil
}
