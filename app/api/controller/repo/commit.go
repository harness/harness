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
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/check"
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
	in.Branch = strings.TrimPrefix(strings.TrimSpace(in.Branch), api.BranchPrefix)
	in.NewBranch = strings.TrimPrefix(strings.TrimSpace(in.NewBranch), api.BranchPrefix)

	if in.Author != nil {
		in.Author.Name = strings.TrimSpace(in.Author.Name)
		in.Author.Email = strings.TrimSpace(in.Author.Email)
	}

	if len(in.Title) > 1024 { // TODO: Check if the title length limit is acceptable.
		return usererror.BadRequest("Commit title is too long.")
	}

	if len(in.Message) > 65536 { // TODO: Check if the message length limit is acceptable.
		return usererror.BadRequest("Commit message is too long.")
	}

	if in.NewBranch != "" {
		if err := check.BranchName(in.NewBranch); err != nil {
			return usererror.BadRequestf("Invalid branch name: %s", err.Error())
		}
	}

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

	var refAction protection.RefAction
	var branchName string
	if in.NewBranch != "" {
		refAction = protection.RefActionCreate
		branchName = in.NewBranch
	} else {
		refAction = protection.RefActionUpdate
		branchName = in.Branch
	}

	rules, isRepoOwner, err := c.fetchBranchRules(ctx, session, repo)
	if err != nil {
		return types.CommitFilesResponse{}, nil, err
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

	mqViolations, err := c.mergeQueueService.BranchInQueueViolations(ctx, repo.ID, branchName)
	if err != nil {
		return types.CommitFilesResponse{}, nil,
			fmt.Errorf("failed to check for merge queue existence: %w", err)
	}

	violations = append(violations, mqViolations...)

	// Content validation is advisory: it is reported on the response and never affects the outcome.
	contentValidations := c.validateActionContent(ctx, repo, in.Actions)

	if in.DryRunRules {
		return types.CommitFilesResponse{
			DryRunRulesOutput: types.DryRunRulesOutput{
				DryRunRules:    true,
				RuleViolations: violations,
			},
			ContentValidations: contentValidations,
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

	// Create API write params based on bypass_rules.
	// If bypass_rules is true, use APIContentBypassRules operation type.
	var writeParams git.WriteParams
	if in.BypassRules {
		writeParams, err = controller.CreateRPCAPIContentBypassRulesWriteParams(
			ctx, c.urlProvider, session, repo,
		)
	} else {
		writeParams, err = controller.CreateRPCAPIContentWriteParams(
			ctx, c.urlProvider, session, repo,
		)
	}
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
		// Push rules run in the pre-receive hook (they need the pushed objects); surface
		// their violations to the API/UI instead of an opaque error.
		if pushViolations, ok := controller.RuleViolationsFromError(err); ok {
			return types.CommitFilesResponse{}, append(violations, pushViolations...), nil
		}
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
		ChangedFiles:       mapChangedFiles(commit.ChangedFiles),
		ContentValidations: contentValidations,
	}, nil, nil
}

// validateActionContent runs the content validators over the files a commit writes. It is advisory and returns
// no error: a validator failure is logged and that file is left unvalidated, so it can never fail a commit.
func (c *Controller) validateActionContent(
	ctx context.Context,
	repo *types.RepositoryCore,
	actions []CommitFileAction,
) []types.ContentValidation {
	var result []types.ContentValidation
	for _, action := range actions {
		path, content, ok := validatableContent(action)
		if !ok || !c.contentValidators.Handles(path) {
			continue
		}

		violations, err := c.contentValidators.Validate(ctx, repo, path, content)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("path", path).Msg("content validation failed, skipping")
			continue
		}
		if len(violations) == 0 {
			continue
		}
		result = append(result, types.ContentValidation{Path: path, Violations: violations})
	}
	return result
}

// validatableContent returns the destination path and decoded content of an action, if it carries content.
func validatableContent(action CommitFileAction) (string, []byte, bool) {
	var payload []byte
	switch action.Encoding {
	case enum.ContentEncodingTypeBase64:
		decoded, err := base64.StdEncoding.DecodeString(action.Payload)
		if err != nil {
			return "", nil, false
		}
		payload = decoded
	case enum.ContentEncodingTypeUTF8:
		fallthrough
	default:
		payload = []byte(action.Payload)
	}

	// Git writes the file at the cleaned path, so claim it by that same path: "docs/../CODEOWNERS" writes
	// CODEOWNERS and must be validated as CODEOWNERS.
	switch action.Action {
	case git.CreateAction, git.UpdateAction:
		return api.CleanUploadFileName(action.Path), payload, true
	case git.MoveAction:
		// A move payload is the new path, optionally followed by a NUL byte and the new content.
		newPath, content, found := bytes.Cut(payload, []byte{0})
		if !found {
			return "", nil, false
		}
		return api.CleanUploadFileName(string(newPath)), content, true
	case git.DeleteAction, git.PatchTextAction:
		return "", nil, false
	default:
		return "", nil, false
	}
}
