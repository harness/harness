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

package check

import (
	"context"
	"fmt"
	"slices"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const maxExcludePayloadKinds = 10

// SummaryInput lists the commits to summarize status checks for.
type SummaryInput struct {
	CommitSHAs          []string `json:"commit_shas" description:"Full commit SHAs. At most 100 are allowed."`
	ExcludePayloadKinds []string `json:"exclude_payload_kinds" description:"Check payload kinds not to count. At most 10."`
}

// Sanitize checks the request size and that every commit SHA is a full hash.
// CommitSHAs is sorted and de-duplicated.
func (in *SummaryInput) Sanitize() error {
	if len(in.CommitSHAs) > request.PerPageMax {
		return usererror.ErrRequestTooLarge
	}
	if len(in.ExcludePayloadKinds) > maxExcludePayloadKinds {
		return usererror.BadRequestf("At most %d payload kinds can be excluded.", maxExcludePayloadKinds)
	}

	for i, commitSHA := range in.CommitSHAs {
		if !git.ValidateCommitSHA(commitSHA) {
			return usererror.BadRequestf("Invalid commit SHA at index %d.", i)
		}
	}

	slices.Sort(in.CommitSHAs)
	in.CommitSHAs = slices.Compact(in.CommitSHAs)

	return nil
}

// ListCheckSummaries returns status check counts for each of the provided commits that has checks.
func (c *Controller) ListCheckSummaries(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *SummaryInput,
) ([]types.CommitCheckSummary, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	return c.listCheckSummaries(ctx, repo.ID, in)
}

func (c *Controller) listCheckSummaries(
	ctx context.Context,
	repoID int64,
	in *SummaryInput,
) ([]types.CommitCheckSummary, error) {
	if in == nil || len(in.CommitSHAs) == 0 {
		return []types.CommitCheckSummary{}, nil
	}
	if err := in.Sanitize(); err != nil {
		return nil, err
	}

	excludePayloadKinds := make([]enum.CheckPayloadKind, len(in.ExcludePayloadKinds))
	for i, kind := range in.ExcludePayloadKinds {
		excludePayloadKinds[i] = enum.CheckPayloadKind(kind)
	}

	checkSummary, err := c.checkStore.ResultSummary(ctx, repoID, in.CommitSHAs, excludePayloadKinds)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch check summary for commits: %w", err)
	}

	summaries := make([]types.CommitCheckSummary, 0, len(checkSummary))
	for _, value := range in.CommitSHAs {
		commitSHA := sha.Must(value)
		summary, ok := checkSummary[commitSHA]
		if !ok {
			continue
		}
		summaries = append(summaries, types.CommitCheckSummary{
			CommitSHA:    commitSHA,
			CheckSummary: summary,
		})
	}

	return summaries, nil
}
