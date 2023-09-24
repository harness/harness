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

package githook

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/githook"
	"github.com/harness/gitness/internal/auth"
	events "github.com/harness/gitness/internal/events/git"
	"github.com/harness/gitness/types"
)

const (
	// gitReferenceNamePrefixBranch is the prefix of references of type branch.
	gitReferenceNamePrefixBranch = "refs/heads/"

	// gitReferenceNamePrefixTag is the prefix of references of type tag.
	gitReferenceNamePrefixTag = "refs/tags/"
)

// PostReceive executes the post-receive hook for a git repository.
func (c *Controller) PostReceive(
	ctx context.Context,
	session *auth.Session,
	repoID int64,
	principalID int64,
	in *githook.PostReceiveInput,
) (*githook.Output, error) {
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	// report ref events (best effort)
	c.reportReferenceEvents(ctx, repoID, principalID, in)

	return &githook.Output{}, nil
}

// reportReferenceEvents is reporting reference events to the event system.
// NOTE: keep best effort for now as it doesn't change the outcome of the git operation.
// TODO: in the future we might want to think about propagating errors so user is aware of events not being triggered.
func (c *Controller) reportReferenceEvents(
	ctx context.Context,
	repoID int64,
	principalID int64,
	in *githook.PostReceiveInput,
) {
	for _, refUpdate := range in.RefUpdates {
		switch {
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixBranch):
			c.reportBranchEvent(ctx, repoID, principalID, refUpdate)
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixTag):
			c.reportTagEvent(ctx, repoID, principalID, refUpdate)
		default:
			// Ignore any other references in post-receive
		}
	}
}

func (c *Controller) reportBranchEvent(
	ctx context.Context,
	repoID int64,
	principalID int64,
	branchUpdate githook.ReferenceUpdate,
) {
	switch {
	case branchUpdate.Old == types.NilSHA:
		c.gitReporter.BranchCreated(ctx, &events.BranchCreatedPayload{
			RepoID:      repoID,
			PrincipalID: principalID,
			Ref:         branchUpdate.Ref,
			SHA:         branchUpdate.New,
		})
	case branchUpdate.New == types.NilSHA:
		c.gitReporter.BranchDeleted(ctx, &events.BranchDeletedPayload{
			RepoID:      repoID,
			PrincipalID: principalID,
			Ref:         branchUpdate.Ref,
			SHA:         branchUpdate.Old,
		})
	default:
		c.gitReporter.BranchUpdated(ctx, &events.BranchUpdatedPayload{
			RepoID:      repoID,
			PrincipalID: principalID,
			Ref:         branchUpdate.Ref,
			OldSHA:      branchUpdate.Old,
			NewSHA:      branchUpdate.New,
			Forced:      false, // TODO: data not available yet
		})
	}
}

func (c *Controller) reportTagEvent(
	ctx context.Context,
	repoID int64,
	principalID int64,
	tagUpdate githook.ReferenceUpdate,
) {
	switch {
	case tagUpdate.Old == types.NilSHA:
		c.gitReporter.TagCreated(ctx, &events.TagCreatedPayload{
			RepoID:      repoID,
			PrincipalID: principalID,
			Ref:         tagUpdate.Ref,
			SHA:         tagUpdate.New,
		})
	case tagUpdate.New == types.NilSHA:
		c.gitReporter.TagDeleted(ctx, &events.TagDeletedPayload{
			RepoID:      repoID,
			PrincipalID: principalID,
			Ref:         tagUpdate.Ref,
			SHA:         tagUpdate.Old,
		})
	default:
		c.gitReporter.TagUpdated(ctx, &events.TagUpdatedPayload{
			RepoID:      repoID,
			PrincipalID: principalID,
			Ref:         tagUpdate.Ref,
			OldSHA:      tagUpdate.Old,
			NewSHA:      tagUpdate.New,
			// tags can only be force updated!
			Forced: true,
		})
	}
}
