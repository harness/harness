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

package repo

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/types/enum"
)

// CommitFileAction holds file operation data.
type CommitFileAction struct {
	Action   gitrpc.FileAction        `json:"action"`
	Path     string                   `json:"path"`
	Payload  string                   `json:"payload"`
	Encoding enum.ContentEncodingType `json:"encoding"`
	SHA      string                   `json:"sha"`
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

func (c *Controller) CommitFiles(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CommitFilesOptions,
) (CommitFilesResponse, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush, false)
	if err != nil {
		return CommitFilesResponse{}, err
	}

	actions := make([]gitrpc.CommitFileAction, len(in.Actions))
	for i, action := range in.Actions {
		var rawPayload []byte
		switch action.Encoding {
		case enum.ContentEncodingTypeBase64:
			rawPayload, err = base64.StdEncoding.DecodeString(action.Payload)
			if err != nil {
				return CommitFilesResponse{}, fmt.Errorf("failed to decode base64 payload: %w", err)
			}
		case enum.ContentEncodingTypeUTF8:
			fallthrough
		default:
			// by default we treat content as is
			rawPayload = []byte(action.Payload)
		}

		actions[i] = gitrpc.CommitFileAction{
			Action:  action.Action,
			Path:    action.Path,
			Payload: rawPayload,
			SHA:     action.SHA,
		}
	}

	writeParams, err := CreateRPCWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	now := time.Now()
	commit, err := c.gitRPCClient.CommitFiles(ctx, &gitrpc.CommitFilesParams{
		WriteParams:   writeParams,
		Title:         in.Title,
		Message:       in.Message,
		Branch:        in.Branch,
		NewBranch:     in.NewBranch,
		Actions:       actions,
		Committer:     rpcIdentityFromPrincipal(bootstrap.NewSystemServiceSession().Principal),
		CommitterDate: &now,
		Author:        rpcIdentityFromPrincipal(session.Principal),
		AuthorDate:    &now,
	})
	if err != nil {
		return CommitFilesResponse{}, err
	}
	return CommitFilesResponse{
		CommitID: commit.CommitID,
	}, nil
}
