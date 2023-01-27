// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package githook

import (
	"context"
	"fmt"
	"strings"

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
	in *types.PostReceiveInput,
) (*types.ServerHookOutput, error) {
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	// report ref events (best effort)
	c.reportReferenceEvents(ctx, in)

	return &types.ServerHookOutput{}, nil
}

// reportReferenceEvents is reporting reference events to the event system.
// NOTE: keep best effort for now as it doesn't change the outcome of the git operation.
// TODO: in the future we might want to think about propagating errors so user is aware of events not being triggered.
func (c *Controller) reportReferenceEvents(ctx context.Context, in *types.PostReceiveInput) {
	for _, refUpdate := range in.RefUpdates {
		switch {
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixBranch):
			c.reportBranchEvent(ctx, in.PrincipalID, in.RepoID, refUpdate)
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixTag):
			c.reportTagEvent(ctx, in.PrincipalID, in.RepoID, refUpdate)
		default:
			// Ignore any other references in post-receive
		}
	}
}

func (c *Controller) reportBranchEvent(ctx context.Context,
	principalID int64, repoID int64, branchUpdate types.ReferenceUpdate) {
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
			// Forced:  false, // TODO: data not available yet
		})
	}
}

func (c *Controller) reportTagEvent(ctx context.Context,
	principalID int64, repoID int64, tagUpdate types.ReferenceUpdate) {
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
