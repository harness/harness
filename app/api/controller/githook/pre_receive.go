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
	"github.com/harness/gitness/app/services/settings"
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

	if in.OperationType == enum.GitOpTypeManageRepo {
		output.Error = ptr.String("Push not allowed for repository management operations")
		return output, nil
	}

	repo, err := c.getRepoCheckAccess(ctx, session, in.RepoID, enum.PermissionRepoPush)
	if err != nil {
		return hook.Output{}, err
	}

	if repo.Type == enum.RepoTypeLinked {
		// Only linked-repository synchronization is allowed to write to linked repositories.
		if in.OperationType != enum.GitOpTypeAPILinkedSync {
			output.Error = ptr.String("Push not allowed to a linked repository")
			return output, nil
		}

		// For linked repositories, we don't check repo settings and protection rules
		return hook.Output{}, nil
	}

	if err := c.limiter.RepoSize(ctx, in.RepoID); err != nil {
		return hook.Output{}, fmt.Errorf(
			"resource limit exceeded: %w", limiter.ErrMaxRepoSizeReached,
		)
	}

	// For API ops that only modify references (branch and tags) without pushing commits
	// controller verifies branch and tag rules.
	// Repository setting and push rules are not applicable for api_refs_only ops.
	if in.OperationType == enum.GitOpTypeAPIRefsOnly {
		return hook.Output{}, nil
	}

	// Git push operations cannot push when repository is not in allowed states
	if in.OperationType == enum.GitOpTypeGitPush && !slices.Contains(allowedRepoStatesForPush, repo.State) {
		output.Error = ptr.String(fmt.Sprintf("Push not allowed when repository is in '%s' state", repo.State))
		return output, nil
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

	// For git push operations, block modification of pullreq references.
	if in.OperationType == enum.GitOpTypeGitPush && c.blockPullReqRefUpdate(refUpdates, repo.State) {
		output.Error = ptr.String(usererror.ErrPullReqRefsCantBeModified.Error())
		return output, nil
	}

	err = c.preReceiveExtender.Extend(ctx, rgit, session, repo, in, &output)
	if err != nil {
		return hook.Output{}, fmt.Errorf("failed to extend pre-receive hook: %w", err)
	}
	if output.Error != nil {
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

	if repo.State != enum.RepoStateActive {
		return hook.Output{}, nil
	}

	// TODO: use store.PrincipalInfoCache once we abstracted principals.
	principal, err := c.principalStore.Find(ctx, in.PrincipalID)
	if err != nil {
		return hook.Output{}, fmt.Errorf(
			"failed to find principal with id %d: %w", in.PrincipalID, err,
		)
	}

	dummySession := &auth.Session{Principal: *principal, Metadata: nil}

	isRepoOwner, err := apiauth.IsRepoOwner(ctx, c.authorizer, dummySession, repo)
	if err != nil {
		return hook.Output{}, fmt.Errorf("failed to determine if user is repo owner: %w", err)
	}

	var rulesViolations []types.RuleViolations

	// check branch and tag protection rules
	refRulesViolations, err := c.checkRefRules(
		ctx, dummySession, repo, refUpdates, protectionRules, isRepoOwner, in.OperationType,
	)
	if err != nil {
		return hook.Output{}, fmt.Errorf("failed to check protection rules: %w", err)
	}
	rulesViolations = append(rulesViolations, refRulesViolations...)

	// check push protection rules and repository settings
	pushRulesViolations, settingsViolations, err := c.checkPushProtection(
		ctx, rgit, repo, principal, isRepoOwner, refUpdates, protectionRules, in, &output,
	)
	if err != nil {
		return hook.Output{}, err
	}
	rulesViolations = append(rulesViolations, pushRulesViolations...)

	processProtectionViolations(&output, rulesViolations, settingsViolations)

	return output, nil
}

type repoSettings struct {
	SecretScanningEnabled   bool
	FileSizeLimit           *int64 // nil if the user hasn't explicitly configured the limit
	PrincipalCommitterMatch bool

	GitLFSEnabled bool
}

// fileSizeLimit uses the user-configured limit when set, or falls back to the system default
// only when no rule-based limits are configured.
func (s repoSettings) fileSizeLimit(hasRuleLimits bool) int64 {
	if s.FileSizeLimit != nil && *s.FileSizeLimit > 0 {
		return *s.FileSizeLimit
	}
	if !hasRuleLimits {
		return settings.DefaultFileSizeLimit
	}
	return 0
}

func (s repoSettings) enabled() bool {
	return s.SecretScanningEnabled || s.FileSizeLimit != nil || s.PrincipalCommitterMatch || s.GitLFSEnabled
}

type repoSettingsViolations struct {
	SecretsFound           bool
	ExceededFileSizeLimit  int64 // 0 = no limit exceeded; >0 = limit value exceeded
	CommitterMismatchFound bool
	UnknownLFSObjectsFound bool
}

func (c *Controller) getRepoSettings(
	ctx context.Context,
	repo *types.RepositoryCore,
) (repoSettings, error) {
	var checks repoSettings

	var err error
	checks.SecretScanningEnabled, err = settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeySecretScanningEnabled,
		settings.DefaultSecretScanningEnabled,
	)
	if err != nil {
		return checks, fmt.Errorf("failed to get repo secret scanning enabled setting: %w", err)
	}

	var fileSizeLimit int64
	explicitlySet, err := c.settings.RepoGet(ctx, repo.ID, settings.KeyFileSizeLimit, &fileSizeLimit)
	if err != nil {
		return checks, fmt.Errorf("failed to get repo file size limit setting: %w", err)
	}
	if explicitlySet {
		checks.FileSizeLimit = &fileSizeLimit
	}

	checks.PrincipalCommitterMatch, err = settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeyPrincipalCommitterMatch,
		settings.DefaultPrincipalCommitterMatch,
	)
	if err != nil {
		return checks, fmt.Errorf("failed to get repo principal committer match setting: %w", err)
	}

	checks.GitLFSEnabled, err = settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeyGitLFSEnabled,
		settings.DefaultGitLFSEnabled,
	)
	if err != nil {
		return checks, fmt.Errorf("failed to get repo Git LFS enabled setting: %w", err)
	}

	return checks, nil
}

// checkPushProtection handles push protection verification for active repositories.
func (c *Controller) checkPushProtection(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	principal *types.Principal,
	isRepoOwner bool,
	refUpdates changedRefs,
	protectionRules []types.RuleInfoInternal,
	in types.GithookPreReceiveInput,
	output *hook.Output,
) ([]types.RuleViolations, *repoSettingsViolations, error) {
	pushProtection := c.protectionManager.FilterPushProtection(protectionRules)
	allowBypass := in.OperationType == enum.GitOpTypeAPIContentBypassRules ||
		in.OperationType == enum.GitOpTypeGitPush
	pushVerifyOut, _, err := pushProtection.PushVerify(
		ctx,
		protection.PushVerifyInput{
			ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
			Actor:              principal,
			IsRepoOwner:        isRepoOwner,
			AllowBypass:        allowBypass,
			RepoID:             repo.ID,
			RepoIdentifier:     repo.Identifier,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify git push objects: %w", err)
	}

	settings, err := c.getRepoSettings(ctx, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get repo settings: %w", err)
	}

	if !settings.enabled() && len(pushVerifyOut.Protections) == 0 {
		// No push protections enabled, skip further processing.
		return []types.RuleViolations{}, nil, nil
	}

	secretsCount, err := c.scanSecrets(
		ctx, rgit, repo,
		settings.SecretScanningEnabled || pushVerifyOut.SecretScanningEnabled,
		in, output,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to scan secrets: %w", err)
	}

	violationsInput := &protection.PushViolationsInput{
		ResolveUserGroupID:      c.userGroupService.ListUserIDsByGroupIDs,
		Actor:                   principal,
		IsRepoOwner:             isRepoOwner,
		AllowBypass:             allowBypass,
		Protections:             pushVerifyOut.Protections,
		FileSizeLimits:          pushVerifyOut.FileSizeLimits,
		PrincipalCommitterMatch: pushVerifyOut.PrincipalCommitterMatch,
		SecretScanningEnabled:   pushVerifyOut.SecretScanningEnabled,
		FoundSecretsCount:       secretsCount,
	}

	var settingsViolations repoSettingsViolations
	if settings.SecretScanningEnabled && secretsCount > 0 {
		settingsViolations.SecretsFound = true
	}

	if err = c.processObjects(
		ctx, rgit,
		repo, principal, refUpdates,
		violationsInput,
		settings, &settingsViolations,
		in, output,
	); err != nil {
		return nil, nil, fmt.Errorf("failed to process pre-receive objects: %w", err)
	}

	var rulesViolations []types.RuleViolations
	if violationsInput.HasViolations() {
		pushViolations, err := pushProtection.Violations(ctx, violationsInput)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to backfill violations: %w", err)
		}

		rulesViolations = pushViolations.Violations
	}

	return rulesViolations, &settingsViolations, nil
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

func (c *Controller) checkRefRules(
	ctx context.Context,
	session *auth.Session,
	repo *types.RepositoryCore,
	refUpdates changedRefs,
	protectionRules []types.RuleInfoInternal,
	isRepoOwner bool,
	opType enum.GitOpType,
) ([]types.RuleViolations, error) {
	branchProtection := c.protectionManager.FilterBranchProtection(protectionRules)
	tagProtection := c.protectionManager.FilterTagProtection(protectionRules)

	var ruleViolations []types.RuleViolations

	// Verify branch and tag rules in pre-receive only for direct git pushes.
	// API-originated operations are handled at the controller layer.
	if opType == enum.GitOpTypeGitPush {
		violations, err := c.checkRefUpdatesRules(
			ctx,
			session,
			repo,
			refUpdates,
			isRepoOwner,
			branchProtection,
			tagProtection,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to verify ref update rules for git push: %w", err)
		}

		ruleViolations = append(ruleViolations, violations...)
	}

	return ruleViolations, nil
}

func (c *Controller) checkRefUpdatesRules(
	ctx context.Context,
	session *auth.Session,
	repo *types.RepositoryCore,
	refUpdates changedRefs,
	isRepoOwner bool,
	branchProtection protection.BranchProtection,
	tagProtection protection.TagProtection,
) ([]types.RuleViolations, error) {
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

func processProtectionViolations(
	output *hook.Output,
	ruleViolations []types.RuleViolations,
	settingsViolations *repoSettingsViolations,
) {
	var outErrorMsg string
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
		outErrorMsg = "blocked by protection rules"
	}

	criticalViolation = false

	if settingsViolations != nil {
		const repoSettingsBlockPrefix = "Push blocked by repository settings: "

		if settingsViolations.SecretsFound {
			output.Messages = append(
				output.Messages,
				repoSettingsBlockPrefix+"Secrets detected.",
			)
			criticalViolation = true
		}
		if settingsViolations.ExceededFileSizeLimit > 0 {
			output.Messages = append(
				output.Messages,
				repoSettingsBlockPrefix+fmt.Sprintf(
					"File size limit of %d bytes exceeded.",
					settingsViolations.ExceededFileSizeLimit,
				),
			)
			criticalViolation = true
		}
		if settingsViolations.CommitterMismatchFound {
			output.Messages = append(
				output.Messages,
				repoSettingsBlockPrefix+"Committer user mismatch detected.",
			)
			criticalViolation = true
		}
		if settingsViolations.UnknownLFSObjectsFound {
			output.Messages = append(
				output.Messages,
				repoSettingsBlockPrefix+"Unknown Git LFS objects detected.",
			)
			criticalViolation = true
		}
	}

	if criticalViolation {
		if outErrorMsg != "" {
			outErrorMsg += ", "
		}
		outErrorMsg += "blocked by repository settings"
	}

	if outErrorMsg != "" {
		output.Error = ptr.String(outErrorMsg)
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
