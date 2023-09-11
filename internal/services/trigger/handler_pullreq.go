// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"

	"github.com/harness/gitness/events"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
)

func (s *Service) handleEventPullReqCreated(ctx context.Context,
	event *events.Event[*pullreqevents.CreatedPayload]) error {
	return events.NewDiscardEventErrorf("not implemented")
}

func (s *Service) handleEventPullReqReopened(ctx context.Context,
	event *events.Event[*pullreqevents.ReopenedPayload]) error {
	return events.NewDiscardEventErrorf("not implemented")
}

func (s *Service) handleEventPullReqBranchUpdated(ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload]) error {
	return events.NewDiscardEventErrorf("not implemented")
}
