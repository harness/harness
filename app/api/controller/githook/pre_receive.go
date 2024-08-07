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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"
)

// PreReceive executes the pre-receive hook for a git repository.
func (c *Controller) PreReceive(
	ctx context.Context,
	rgit RestrictedGIT,
	session *auth.Session,
	in types.GithookPreReceiveInput,
) (hook.Output, error) {
	output := hook.Output{}

	repo, err := c.getRepoCheckAccess(ctx, session, in.RepoID, enum.PermissionRepoPush)
	if err != nil {
		return hook.Output{}, err
	}

	if !in.Internal && repo.State != enum.RepoStateActive && repo.State != enum.RepoStateMigrateGitPush {
		output.Error = ptr.String("Push not allowed in the current repository state")
		return output, nil
	}

	if err := c.limiter.RepoSize(ctx, in.RepoID); err != nil {
		return hook.Output{}, fmt.Errorf(
			"resource limit exceeded: %w",
			limiter.ErrMaxRepoSizeReached)
	}

	refUpdates := groupRefsByAction(in.RefUpdates)

	if slices.Contains(refUpdates.branches.deleted, repo.DefaultBranch) {
		// Default branch mustn't be deleted.
		output.Error = ptr.String(usererror.ErrDefaultBranchCantBeDeleted.Error())
		return output, nil
	}

	// For external calls (git pushes) block modification of pullreq references.
	if !in.Internal && c.blockPullReqRefUpdate(refUpdates, repo.State) {
		output.Error = ptr.String(usererror.ErrPullReqRefsCantBeModified.Error())
		return output, nil
	}

	// For internal calls - through the application interface (API) - no need to verify protection rules.
	if !in.Internal && repo.State == enum.RepoStateActive {
		// TODO: use store.PrincipalInfoCache once we abstracted principals.
		principal, err := c.principalStore.Find(ctx, in.PrincipalID)
		if err != nil {
			return hook.Output{}, fmt.Errorf("failed to find inner principal with id %d: %w", in.PrincipalID, err)
		}

		dummySession := &auth.Session{Principal: *principal, Metadata: nil}

		err = c.checkProtectionRules(ctx, dummySession, repo, refUpdates, &output)
		if err != nil {
			return hook.Output{}, fmt.Errorf("failed to check protection rules: %w", err)
		}
	}

	err = c.scanSecrets(ctx, rgit, repo, in, &output)
	if err != nil {
		return hook.Output{}, err
	}

	err = c.preReceiveExtender.Extend(ctx, rgit, session, repo, in, &output)
	if err != nil {
		return hook.Output{}, fmt.Errorf("failed to extend pre-receive hook: %w", err)
	}

	err = c.checkFileSizeLimit(ctx, rgit, repo, in, &output)
	if output.Error != nil {
		return output, nil
	}
	if err != nil {
		return hook.Output{}, err
	}

	return output, nil
}

func (c *Controller) blockPullReqRefUpdate(refUpdates changedRefs, state enum.RepoState) bool {
	if state == enum.RepoStateMigrateGitPush {
		return false
	}

	fn := func(ref string) bool {
		return strings.HasPrefix(ref, gitReferenceNamePullReq)
	}

	return slices.ContainsFunc(refUpdates.other.created, fn) ||
		slices.ContainsFunc(refUpdates.other.deleted, fn) ||
		slices.ContainsFunc(refUpdates.other.updated, fn)
}

func (c *Controller) checkProtectionRules(
	ctx context.Context,
	session *auth.Session,
	repo *types.Repository,
	refUpdates changedRefs,
	output *hook.Output,
) error {
	isRepoOwner, err := apiauth.IsRepoOwner(ctx, c.authorizer, session, repo)
	if err != nil {
		return fmt.Errorf("failed to determine if user is repo owner: %w", err)
	}

	protectionRules, err := c.protectionManager.ForRepository(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	var ruleViolations []types.RuleViolations
	var errCheckAction error

	checkAction := func(refAction protection.RefAction, refType protection.RefType, names []string) {
		if errCheckAction != nil || len(names) == 0 {
			return
		}

		violations, err := protectionRules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
			Actor:       &session.Principal,
			AllowBypass: true,
			IsRepoOwner: isRepoOwner,
			Repo:        repo,
			RefAction:   refAction,
			RefType:     refType,
			RefNames:    names,
		})
		if err != nil {
			errCheckAction = fmt.Errorf("failed to verify protection rules for git push: %w", err)
			return
		}

		ruleViolations = append(ruleViolations, violations...)
	}

	checkAction(protection.RefActionCreate, protection.RefTypeBranch, refUpdates.branches.created)
	checkAction(protection.RefActionDelete, protection.RefTypeBranch, refUpdates.branches.deleted)
	checkAction(protection.RefActionUpdate, protection.RefTypeBranch, refUpdates.branches.updated)

	if errCheckAction != nil {
		return errCheckAction
	}

	var criticalViolation bool

	for _, ruleViolation := range ruleViolations {
		criticalViolation = criticalViolation || ruleViolation.IsCritical()
		for _, violation := range ruleViolation.Violations {
			var message string
			if ruleViolation.Bypassed {
				message = fmt.Sprintf("Bypassed rule %q: %s", ruleViolation.Rule.Identifier, violation.Message)
			} else {
				message = fmt.Sprintf("Rule %q violation: %s", ruleViolation.Rule.Identifier, violation.Message)
			}
			output.Messages = append(output.Messages, message)
		}
	}

	if criticalViolation {
		output.Error = ptr.String("Blocked by protection rules.")
	}

	return nil
}

type changes struct {
	created []string
	deleted []string
	updated []string
}

func (c *changes) groupByAction(refUpdate hook.ReferenceUpdate, name string) {
	switch {
	case refUpdate.Old.IsNil():
		c.created = append(c.created, name)
	case refUpdate.New.IsNil():
		c.deleted = append(c.deleted, name)
	default:
		c.updated = append(c.updated, name)
	}
}

type changedRefs struct {
	branches changes
	tags     changes
	other    changes
}

func groupRefsByAction(refUpdates []hook.ReferenceUpdate) (c changedRefs) {
	for _, refUpdate := range refUpdates {
		switch {
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixBranch):
			branchName := refUpdate.Ref[len(gitReferenceNamePrefixBranch):]
			c.branches.groupByAction(refUpdate, branchName)
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixTag):
			tagName := refUpdate.Ref[len(gitReferenceNamePrefixTag):]
			c.tags.groupByAction(refUpdate, tagName)
		default:
			c.other.groupByAction(refUpdate, refUpdate.Ref)
		}
	}
	return
}

func loggingWithRefUpdate(refUpdate hook.ReferenceUpdate) func(c zerolog.Context) zerolog.Context {
	return func(c zerolog.Context) zerolog.Context {
		return c.Str("ref", refUpdate.Ref).Str("old_sha", refUpdate.Old.String()).Str("new_sha", refUpdate.New.String())
	}
}
