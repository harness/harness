// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// CreateCommitTagInput used for tag creation apis.
type CreateCommitTagInput struct {
	Name string `json:"name"`
	// Target is the commit (or points to the commit) the new tag will be pointing to.
	// If no target is provided, the tag points to the same commit as the default branch of the repo.
	Target string `json:"target"`

	// Message is the optional message the tag will be created with - if the message is empty
	// the tag will be lightweight, otherwise it'll be annotated.
	Message string `json:"message"`
}

// CreateCommitTag creates a new tag for a repo.
func (c *Controller) CreateCommitTag(ctx context.Context, session *auth.Session,
	repoRef string, in *CreateCommitTagInput) (*CommitTag, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoPush, false); err != nil {
		return nil, err
	}

	// set target to default branch in case no branch or commit was provided
	if in.Target == "" {
		in.Target = repo.DefaultBranch
	}

	writeParams, err := CreateRPCWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	now := time.Now()
	rpcOut, err := c.gitRPCClient.CreateCommitTag(ctx, &gitrpc.CreateCommitTagParams{
		WriteParams: writeParams,
		Name:        in.Name,
		Target:      in.Target,
		Message:     in.Message,
		Tagger:      rpcIdentityFromPrincipal(session.Principal),
		TaggerDate:  &now,
	})

	if err != nil {
		return nil, err
	}
	commitTag, err := mapCommitTag(rpcOut.CommitTag)

	if err != nil {
		return nil, fmt.Errorf("failed to map tag received from service output: %w", err)
	}
	return &commitTag, nil
}
