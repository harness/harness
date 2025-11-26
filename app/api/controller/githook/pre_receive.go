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

// allowedRepoStatesForPush lists repository states that git push is allowed for internal and external calls.
var allowedRepoStatesForPush = []enum.RepoState{enum.RepoStateActive, enum.RepoStateMigrateGitPush}

// PreReceive executes the pre-receive hook for a git repository.
func (c *Controller) PreReceive(
	ctx context.Context,
	rgit RestrictedGIT,
	session *auth.Session,
	in types.GithookPreReceiveInput,
) (hook.Output, error) {
	output := hook.Output{}
	defer func() {
		logOutputFor(ctx, "pre-receive", output)
	}()

	repo, err := c.getRepoCheckAccess(ctx, session, in.RepoID, enum.PermissionRepoPush)
	if err != nil {
		return hook.Output{}, err
	}

	if !in.Internal && !slices.Contains(allowedRepoStatesForPush, repo.State) {
		output.Error = ptr.String(fmt.Sprintf("Push not allowed when repository is in '%s' state", repo.State))
		return output, nil
	}

	if err := c.limiter.RepoSize(ctx, in.RepoID); err != nil {
		return hook.Output{}, fmt.Errorf(
			"resource limit exceeded: %w", limiter.ErrMaxRepoSizeReached,
		)
	}

	forced := make([]bool, len(in.RefUpdates))
	for i, refUpdate := range in.RefUpdates {
		forced[i], err = isForcePush(
			ctx, rgit, repo.GitUID, in.Environment.AlternateObjectDirs, refUpdate,
		)
		if err != nil {
			return hook.Output{}, fmt.Errorf("failed to check branch ancestor: %w", err)
		}
	}

	refUpdates := groupRefsByAction(in.RefUpdates, forced)

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

	protectionRules, err := c.protectionManager.ListRepoRules(
		ctx, repo.ID, protection.TypeBranch, protection.TypeTag, protection.TypePush,
	)
	if err != nil {
		return hook.Output{}, fmt.Errorf(
			"failed to fetch protection rules for the repository: %w", err,
		)
	}

	var principal *types.Principal
	repoActive := repo.State == enum.RepoStateActive
	if repoActive {
		// TODO: use store.PrincipalInfoCache once we abstracted principals.
		principal, err = c.principalStore.Find(ctx, in.PrincipalID)
		if err != nil {
			return hook.Output{}, fmt.Errorf("failed to find inner principal with id %d: %w", in.PrincipalID, err)
		}
	}

	var ruleViolations []types.RuleViolations
	var isRepoOwner bool
	// For internal calls - through the application interface (API) - no need to verify protection rules.
	if !in.Internal && repoActive {
		dummySession := &auth.Session{Principal: *principal, Metadata: nil}
		isRepoOwner, err = apiauth.IsRepoOwner(ctx, c.authorizer, dummySession, repo)
		if err != nil {
			return hook.Output{}, fmt.Errorf("failed to determine if user is repo owner: %w", err)
		}

		ruleViolations, err = c.checkProtectionRules(
			ctx, dummySession, repo, refUpdates, protectionRules, isRepoOwner,
		)
		if err != nil {
			return hook.Output{}, fmt.Errorf("failed to check protection rules: %w", err)
		}
		if output.Error != nil {
			return output, nil
		}
	}

	err = c.preReceiveExtender.Extend(ctx, rgit, session, repo, in, &output)
	if err != nil {
		return hook.Output{}, fmt.Errorf("failed to extend pre-receive hook: %w", err)
	}
	if output.Error != nil {
		return output, nil
	}

	if repoActive {
		// check secret scanning apart from push rules as it is enabled in repository settings.
		err = c.scanSecrets(ctx, rgit, repo, false, nil, in, &output)
		if err != nil {
			return hook.Output{}, fmt.Errorf("failed to scan secrets: %w", err)
		}
		if output.Error != nil {
			return output, nil
		}

		violations, err := c.processPushProtection(
			ctx, rgit, repo, principal, isRepoOwner, refUpdates, protectionRules, in, &output,
		)
		if err != nil {
			return hook.Output{}, err
		}
		ruleViolations = append(ruleViolations, violations...)

		processRuleViolations(&output, ruleViolations)
	}

	return output, nil
}

// processPushProtection handles push protection verification for active repositories.
func (c *Controller) processPushProtection(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	principal *types.Principal,
	isRepoOwner bool,
	refUpdates changedRefs,
	protectionRules []types.RuleInfoInternal,
	in types.GithookPreReceiveInput,
	output *hook.Output,
) ([]types.RuleViolations, error) {
	pushProtection := c.protectionManager.FilterCreatePushProtection(protectionRules)
	out, _, err := pushProtection.PushVerify(
		ctx,
		protection.PushVerifyInput{
			ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
			Actor:              principal,
			IsRepoOwner:        isRepoOwner,
			RepoID:             repo.ID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify git objects: %w", err)
	}

	violationsInput := &protection.PushViolationsInput{
		ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
		Actor:              principal,
		IsRepoOwner:        isRepoOwner,
		Protections:        out.Protections,
	}

	err = c.scanSecrets(ctx, rgit, repo, out.SecretScanningEnabled, violationsInput, in, output)
	if err != nil {
		return nil, fmt.Errorf("failed to scan secrets: %w", err)
	}

	if err = c.processObjects(
		ctx, rgit,
		repo, principal, refUpdates,
		out.FileSizeLimit, out.PrincipalCommitterMatch, violationsInput,
		in, output,
	); err != nil {
		return nil, fmt.Errorf("failed to process pre-receive objects: %w", err)
	}

	var violations []types.RuleViolations
	if violationsInput.HasViolations() {
		pushViolations, err := pushProtection.Violations(ctx, violationsInput)
		if err != nil {
			return nil, fmt.Errorf("failed to backfill violations: %w", err)
		}

		violations = pushViolations.Violations
	}

	return violations, nil
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
		slices.ContainsFunc(refUpdates.other.updated, fn) ||
		slices.ContainsFunc(refUpdates.other.forced, fn)
}

func (c *Controller) checkProtectionRules(
	ctx context.Context,
	session *auth.Session,
	repo *types.RepositoryCore,
	refUpdates changedRefs,
	protectionRules []types.RuleInfoInternal,
	isRepoOwner bool,
) ([]types.RuleViolations, error) {
	branchProtection := c.protectionManager.FilterCreateBranchProtection(protectionRules)
	tagProtection := c.protectionManager.FilterCreateTagProtection(protectionRules)

	var ruleViolations []types.RuleViolations
	var errCheckAction error

	//nolint:unparam
	checkAction := func(
		refProtection protection.RefProtection,
		refAction protection.RefAction,
		refType protection.RefType,
		names []string,
	) {
		if errCheckAction != nil || len(names) == 0 {
			return
		}

		violations, err := refProtection.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
			ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
			Actor:              &session.Principal,
			AllowBypass:        true,
			IsRepoOwner:        isRepoOwner,
			Repo:               repo,
			RefAction:          refAction,
			RefType:            refType,
			RefNames:           names,
		})
		if err != nil {
			errCheckAction = fmt.Errorf("failed to verify protection rules for git push: %w", err)
			return
		}

		ruleViolations = append(ruleViolations, violations...)
	}

	checkAction(
		branchProtection, protection.RefActionCreate,
		protection.RefTypeBranch, refUpdates.branches.created,
	)
	checkAction(
		branchProtection, protection.RefActionDelete,
		protection.RefTypeBranch, refUpdates.branches.deleted,
	)
	checkAction(
		branchProtection, protection.RefActionUpdate,
		protection.RefTypeBranch, refUpdates.branches.updated,
	)
	checkAction(
		branchProtection, protection.RefActionUpdateForce,
		protection.RefTypeBranch, refUpdates.branches.forced,
	)

	checkAction(
		tagProtection, protection.RefActionCreate,
		protection.RefTypeTag, refUpdates.tags.created,
	)
	checkAction(
		tagProtection, protection.RefActionDelete,
		protection.RefTypeTag, refUpdates.tags.deleted,
	)
	checkAction(
		tagProtection, protection.RefActionUpdateForce,
		protection.RefTypeTag, refUpdates.tags.forced,
	)

	if errCheckAction != nil {
		return nil, errCheckAction
	}

	return ruleViolations, nil
}

func processRuleViolations(
	output *hook.Output,
	ruleViolations []types.RuleViolations,
) {
	if len(ruleViolations) == 0 {
		return
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
}

type changes struct {
	created []string
	deleted []string
	updated []string
	forced  []string
}

func (c *changes) groupByAction(
	refUpdate hook.ReferenceUpdate,
	name string,
	forced bool,
) {
	switch {
	case refUpdate.Old.IsNil():
		c.created = append(c.created, name)
	case refUpdate.New.IsNil():
		c.deleted = append(c.deleted, name)
	case forced:
		c.forced = append(c.forced, name)
	default:
		c.updated = append(c.updated, name)
	}
}

type changedRefs struct {
	branches changes
	tags     changes
	other    changes
}

func (c *changedRefs) hasOnlyDeletedBranches() bool {
	if len(c.branches.created) > 0 || len(c.branches.updated) > 0 || len(c.branches.forced) > 0 {
		return false
	}
	return true
}

func isBranch(ref string) bool {
	return strings.HasPrefix(ref, gitReferenceNamePrefixBranch)
}

func isTag(ref string) bool {
	return strings.HasPrefix(ref, gitReferenceNamePrefixTag)
}

func groupRefsByAction(refUpdates []hook.ReferenceUpdate, forced []bool) (c changedRefs) {
	for i, refUpdate := range refUpdates {
		switch {
		case isBranch(refUpdate.Ref):
			branchName := refUpdate.Ref[len(gitReferenceNamePrefixBranch):]
			c.branches.groupByAction(refUpdate, branchName, forced[i])
		case isTag(refUpdate.Ref):
			tagName := refUpdate.Ref[len(gitReferenceNamePrefixTag):]
			c.tags.groupByAction(refUpdate, tagName, forced[i])
		default:
			c.other.groupByAction(refUpdate, refUpdate.Ref, false)
		}
	}
	return
}

func loggingWithRefUpdate(refUpdate hook.ReferenceUpdate) func(c zerolog.Context) zerolog.Context {
	return func(c zerolog.Context) zerolog.Context {
		return c.Str("ref", refUpdate.Ref).Str("old_sha", refUpdate.Old.String()).Str("new_sha", refUpdate.New.String())
	}
}
