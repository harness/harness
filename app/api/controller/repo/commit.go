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
	"cmp"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// CommitFileAction holds file operation data.
type CommitFileAction struct {
	Action   git.FileAction           `json:"action"`
	Path     string                   `json:"path"`
	Payload  string                   `json:"payload"`
	Encoding enum.ContentEncodingType `json:"encoding"`

	// SHA can be used for optimistic locking of an update action (Optional).
	// The provided value is compared against the latest sha of the file that's being updated.
	// If the SHA doesn't match, the update fails.
	// WARNING: If no SHA is provided, the update action will blindly overwrite the file's content.
	SHA sha.SHA `json:"sha"`
}

// CommitFilesOptions holds the data for file operations.
type CommitFilesOptions struct {
	Title     string             `json:"title"`
	Message   string             `json:"message"`
	Branch    string             `json:"branch"`
	NewBranch string             `json:"new_branch"`
	Actions   []CommitFileAction `json:"actions"`
	Author    *git.Identity      `json:"author"`

	DryRunRules bool `json:"dry_run_rules"`
	BypassRules bool `json:"bypass_rules"`
}

func (in *CommitFilesOptions) Sanitize() error {
	in.Title = strings.TrimSpace(in.Title)
	in.Message = strings.TrimSpace(in.Message)

	// TODO: Validate title and message length.

	return nil
}

func mapChangedFiles(files []git.FileReference) []types.FileReference {
	changedFiles := make([]types.FileReference, len(files))
	for i, file := range files {
		changedFiles[i] = types.FileReference{
			Path: file.Path,
			SHA:  file.SHA,
		}
	}
	return changedFiles
}

func (c *Controller) CommitFiles(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CommitFilesOptions,
) (types.CommitFilesResponse, []types.RuleViolations, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return types.CommitFilesResponse{}, nil, err
	}

	if err := in.Sanitize(); err != nil {
		return types.CommitFilesResponse{}, nil, err
	}

	rules, isRepoOwner, err := c.fetchBranchRules(ctx, session, repo)
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
		ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
		Actor:              &session.Principal,
		AllowBypass:        in.BypassRules,
		IsRepoOwner:        isRepoOwner,
		Repo:               repo,
		RefAction:          refAction,
		RefType:            protection.RefTypeBranch,
		RefNames:           []string{branchName},
	})
	if err != nil {
		return types.CommitFilesResponse{}, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if in.DryRunRules {
		return types.CommitFilesResponse{
			DryRunRulesOutput: types.DryRunRulesOutput{
				DryRunRules:    true,
				RuleViolations: violations,
			},
		}, nil, nil
	}

	if protection.IsCritical(violations) {
		return types.CommitFilesResponse{}, violations, nil
	}

	actions := make([]git.CommitFileAction, len(in.Actions))
	for i, action := range in.Actions {
		var rawPayload []byte
		switch action.Encoding {
		case enum.ContentEncodingTypeBase64:
			rawPayload, err = base64.StdEncoding.DecodeString(action.Payload)
			if err != nil {
				return types.CommitFilesResponse{}, nil, errors.Internal(err, "failed to decode base64 payload")
			}
		case enum.ContentEncodingTypeUTF8:
			fallthrough
		default:
			// by default we treat content as is
			rawPayload = []byte(action.Payload)
		}

		actions[i] = git.CommitFileAction{
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
	commit, err := c.git.CommitFiles(ctx, &git.CommitFilesParams{
		WriteParams:   writeParams,
		Message:       git.CommitMessage(in.Title, in.Message),
		Branch:        in.Branch,
		NewBranch:     in.NewBranch,
		Actions:       actions,
		Committer:     identityFromPrincipal(bootstrap.NewSystemServiceSession().Principal),
		CommitterDate: &now,
		Author:        cmp.Or(in.Author, identityFromPrincipal(session.Principal)),
		AuthorDate:    &now,
	})
	if err != nil {
		return types.CommitFilesResponse{}, nil, err
	}

	if protection.IsBypassed(violations) {
		err = c.auditService.Log(ctx,
			session.Principal,
			audit.NewResource(
				audit.ResourceTypeRepository,
				repo.Identifier,
				audit.RepoPath,
				repo.Path,
				audit.BypassAction,
				audit.BypassActionCommitted,
				audit.BypassedResourceType,
				audit.BypassedResourceTypeCommit,
				audit.BypassedResourceName,
				commit.CommitID.String(),
				audit.ResourceName,
				fmt.Sprintf(
					audit.BypassSHALabelFormat,
					repo.Identifier,
					commit.CommitID.String()[0:6],
				),
			),
			audit.ActionBypassed,
			paths.Parent(repo.Path),
			audit.WithNewObject(audit.CommitObject{
				CommitSHA:      commit.CommitID.String(),
				RepoPath:       repo.Path,
				RuleViolations: violations,
			}),
		)
	}
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for commit operation: %s", err)
	}

	return types.CommitFilesResponse{
		CommitID: commit.CommitID,
		DryRunRulesOutput: types.DryRunRulesOutput{
			RuleViolations: violations,
		},
		ChangedFiles: mapChangedFiles(commit.ChangedFiles),
	}, nil, nil
}
