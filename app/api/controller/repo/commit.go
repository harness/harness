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

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CommitFileAction holds file operation data.
type CommitFileAction struct {
	Action   gitrpc.FileAction        `json:"action"`
	Path     string                   `json:"path"`
	Payload  string                   `json:"payload"`
	Encoding enum.ContentEncodingType `json:"encoding"`

	// SHA can be used for optimistic locking of an update action (Optional).
	// The provided value is compared against the latest sha of the file that's being updated.
	// If the SHA doesn't match, the update fails.
	// WARNING: If no SHA is provided, the update action will blindly overwrite the file's content.
	SHA string `json:"sha"`
}

// CommitFilesOptions holds the data for file operations.
type CommitFilesOptions struct {
	Title     string             `json:"title"`
	Message   string             `json:"message"`
	Branch    string             `json:"branch"`
	NewBranch string             `json:"new_branch"`
	Actions   []CommitFileAction `json:"actions"`
}

func (c *Controller) CommitFiles(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CommitFilesOptions,
) (types.CommitFilesResponse, []types.RuleViolations, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush, false)
	if err != nil {
		return types.CommitFilesResponse{}, nil, err
	}

	rules, isRepoOwner, err := c.fetchRules(ctx, session, repo)
	if err != nil {
		return types.CommitFilesResponse{}, nil, err
	}

	var refAction protection.RefAction
	var branchName string
	if in.NewBranch != "" {
		refAction = protection.RefActionCreate
		branchName = in.NewBranch
	} else {
		refAction = protection.RefActionUpdate
		branchName = in.Branch
	}

	violations, err := rules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
		Actor:       &session.Principal,
		AllowBypass: true,
		IsRepoOwner: isRepoOwner,
		Repo:        repo,
		RefAction:   refAction,
		RefType:     protection.RefTypeBranch,
		RefNames:    []string{branchName},
	})
	if err != nil {
		return types.CommitFilesResponse{}, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if protection.IsCritical(violations) {
		return types.CommitFilesResponse{}, violations, nil
	}

	actions := make([]gitrpc.CommitFileAction, len(in.Actions))
	for i, action := range in.Actions {
		var rawPayload []byte
		switch action.Encoding {
		case enum.ContentEncodingTypeBase64:
			rawPayload, err = base64.StdEncoding.DecodeString(action.Payload)
			if err != nil {
				return types.CommitFilesResponse{}, nil, fmt.Errorf("failed to decode base64 payload: %w", err)
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

	// Create internal write params. Note: This will skip the pre-commit protection rules check.
	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return types.CommitFilesResponse{}, nil, fmt.Errorf("failed to create RPC write params: %w", err)
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
		return types.CommitFilesResponse{}, nil, err
	}

	return types.CommitFilesResponse{
		CommitID: commit.CommitID,
	}, nil, nil
}
