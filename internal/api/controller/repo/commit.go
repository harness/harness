// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// CommitFileAction holds file operation data.
type CommitFileAction struct {
	Action   gitrpc.FileAction `json:"action"`
	Path     string            `json:"path"`
	Payload  string            `json:"payload"`
	Encoding string            `json:"encoding"`
	SHA      string            `json:"sha"`
}

// CommitFilesOptions holds the data for file operations.
type CommitFilesOptions struct {
	Title     string             `json:"title"`
	Message   string             `json:"message"`
	Branch    string             `json:"branch"`
	NewBranch string             `json:"new_branch"`
	Actions   []CommitFileAction `json:"actions"`
}

// CommitFilesResponse holds commit id.
type CommitFilesResponse struct {
	CommitID string `json:"commit_id"`
}

func (c *Controller) CommitFiles(ctx context.Context, session *auth.Session,
	repoRef string, in *CommitFilesOptions) (CommitFilesResponse, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return CommitFilesResponse{}, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return CommitFilesResponse{}, err
	}

	actions := make([]gitrpc.CommitFileAction, len(in.Actions))
	for i, action := range in.Actions {
		actions[i] = gitrpc.CommitFileAction{
			Action:   action.Action,
			Path:     action.Path,
			Payload:  []byte(action.Payload),
			Encoding: action.Encoding,
			SHA:      action.SHA,
		}
	}

	writeParams, err := CreateRPCWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	commit, err := c.gitRPCClient.CommitFiles(ctx, &gitrpc.CommitFilesParams{
		WriteParams: writeParams,
		Title:       in.Title,
		Message:     in.Message,
		Branch:      in.Branch,
		NewBranch:   in.NewBranch,
		Author: gitrpc.Identity{
			Name:  session.Principal.DisplayName,
			Email: session.Principal.Email,
		},
		Actions: actions,
	})
	if err != nil {
		return CommitFilesResponse{}, err
	}
	return CommitFilesResponse{
		CommitID: commit.CommitID,
	}, nil
}
