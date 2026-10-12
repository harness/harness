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

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
)

// ErrReviewerLimitExceeded is the user facing error for exceeding the pull request
// reviewer limit (combined count of principal and user group reviewers).
var ErrReviewerLimitExceeded = usererror.BadRequestf(
	"Pull request can't have more than %d reviewers (users and user groups combined).",
	types.MaxPullReqReviewers,
)

// CheckReviewerLimit returns a user facing error if the pull request can't take another
// reviewer. The limit is for the combined count of principal and user group reviewers.
// Call it inside the transaction which adds the reviewer to keep the limit tight.
//
// It locks the pull request row first: on Postgres, under the default READ COMMITTED
// isolation, two concurrent calls could otherwise both pass the count check before either
// commits its insert. The lock serializes them so the second call's count reflects the
// first call's insert.
func (s *Service) CheckReviewerLimit(ctx context.Context, prID int64) error {
	if _, err := s.pullreqStore.FindWithLock(ctx, prID); err != nil {
		return fmt.Errorf("failed to lock pull request: %w", err)
	}

	reviewerCount, err := s.reviewerStore.Count(ctx, prID)
	if err != nil {
		return fmt.Errorf("failed to count reviewers: %w", err)
	}

	userGroupReviewerCount, err := s.userGroupReviewerStore.Count(ctx, prID)
	if err != nil {
		return fmt.Errorf("failed to count user group reviewers: %w", err)
	}

	if reviewerCount+userGroupReviewerCount >= types.MaxPullReqReviewers {
		return ErrReviewerLimitExceeded
	}

	return nil
}
