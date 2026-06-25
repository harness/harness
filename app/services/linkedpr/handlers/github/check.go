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

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	checkevents "github.com/harness/gitness/app/events/check"
	"github.com/harness/gitness/app/services/linkedpr"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// CheckHandler upserts external CI status checks received via GitHub.
// check_run / status webhook events into the gitness CheckStore.
type CheckHandler struct {
	checkStore     store.CheckStore
	eventReporter  *checkevents.Reporter
	authorResolver linkedpr.AuthorResolver
}

func NewCheckHandler(
	checkStore store.CheckStore,
	eventReporter *checkevents.Reporter,
	authorResolver linkedpr.AuthorResolver,
) *CheckHandler {
	return &CheckHandler{
		checkStore:     checkStore,
		eventReporter:  eventReporter,
		authorResolver: authorResolver,
	}
}

func (h *CheckHandler) Handle(
	ctx context.Context,
	ev *linkedpr.Event,
	payload linkedpr.CheckPayload,
	linkedRepo *types.LinkedRepo,
) error {
	reporterPrincipalID, err := h.authorResolver.Resolve(ctx, ev)
	if err != nil {
		return fmt.Errorf("resolve reporter: %w", err)
	}

	now := time.Now().UnixMilli()

	for _, entry := range payload.Checks {
		if entry.SHA == "" {
			log.Ctx(ctx).Warn().
				Str("identifier", entry.Identifier).
				Msg("linkedpr: check entry missing SHA; skipping")
			continue
		}
		if entry.Identifier == "" {
			log.Ctx(ctx).Warn().
				Str("sha", entry.SHA).
				Msg("linkedpr: check entry missing identifier; skipping")
			continue
		}
		if entry.Link == "" {
			log.Ctx(ctx).Warn().
				Str("identifier", entry.Identifier).
				Str("sha", entry.SHA).
				Msg("linkedpr: check entry missing link; skipping")
			continue
		}

		status := mapCheckStatus(entry.Status, entry.Conclusion)

		check := &types.Check{
			CreatedBy:  reporterPrincipalID,
			Created:    now,
			Updated:    now,
			RepoID:     linkedRepo.RepoID,
			CommitSHA:  entry.SHA,
			Identifier: entry.Identifier,
			Status:     status,
			Link:       entry.Link,
			Started:    parseTimeMillis(entry.Started),
			Ended:      parseTimeMillis(entry.Completed),
			Metadata:   json.RawMessage(`{}`),
			Payload: types.CheckPayload{
				Kind: enum.CheckPayloadKindEmpty,
				Data: json.RawMessage(`{}`),
			},
		}

		if err := h.checkStore.Upsert(ctx, check); err != nil {
			return fmt.Errorf("upsert check %q sha=%s: %w", entry.Identifier, entry.SHA, err)
		}

		h.eventReporter.Reported(ctx, &checkevents.ReportedPayload{
			Base: checkevents.Base{
				RepoID: linkedRepo.RepoID,
				SHA:    entry.SHA,
			},
			Identifier: entry.Identifier,
			Status:     status,
		})
	}
	return nil
}

// mapCheckStatus converts GitHub check_run status + conclusion into a gitness CheckStatus.
// GitHub lifecycle: queued → in_progress → completed (conclusion tells the result).
func mapCheckStatus(status, conclusion string) enum.CheckStatus {
	// conclusion is the authoritative terminal signal — non-empty only when the
	// check has reached a final state. Check it first to handle all GitHub
	// Checks API conclusion values plus legacy Statuses API error/pending states.
	switch conclusion {
	case "success", "neutral", "skipped":
		return enum.CheckStatusSuccess
	case "failure", "timed_out", "action_required", "cancelled":
		return enum.CheckStatusFailure
	case "startup_failure", "error":
		return enum.CheckStatusError
	case "pending":
		return enum.CheckStatusPending
	case "stale":
		// stale means a required check was cleared by a maintainer — treat as pending.
		return enum.CheckStatusPending
	}

	// No conclusion yet — check is still in flight. Use the normalized status.
	// Any terminal check (completed) always has a non-empty conclusion per the
	// GitHub API contract, so only non-terminal values are meaningful here.
	switch status {
	case "running", "in_progress":
		return enum.CheckStatusRunning
	default:
		// queued, pending, Unknown, or anything unexpected — treat as pending.
		return enum.CheckStatusPending
	}
}

// parseTimeMillis parses an RFC 3339 timestamp string and returns Unix milliseconds.
func parseTimeMillis(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}
