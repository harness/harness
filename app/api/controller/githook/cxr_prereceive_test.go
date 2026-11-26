// Copyright 2026 Harness, Inc.
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
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func (e *cxrEnv) preInput(op enum.GitOpType, updates ...hook.ReferenceUpdate) types.GithookPreReceiveInput {
	return types.GithookPreReceiveInput{
		GithookInputBase: types.GithookInputBase{RepoID: e.repo.ID, PrincipalID: e.admin.ID, OperationType: op},
		PreReceiveInput:  hook.PreReceiveInput{RefUpdates: updates},
	}
}

func cxrCreate(ref string) hook.ReferenceUpdate {
	return hook.ReferenceUpdate{Ref: ref, Old: sha.Nil, New: cxrSHA1}
}

func cxrDelete(ref string) hook.ReferenceUpdate {
	return hook.ReferenceUpdate{Ref: ref, Old: cxrSHA1, New: sha.Nil}
}

func cxrUpdate(ref string) hook.ReferenceUpdate {
	return hook.ReferenceUpdate{Ref: ref, Old: cxrSHA1, New: cxrSHA2}
}

func (e *cxrEnv) setSetting(ctx context.Context, t *testing.T, key settings.Key, value any) {
	t.Helper()
	require.NoError(t, e.settings.Set(ctx, enum.SettingsScopeRepo, e.repo.ID, key, value))
}

func TestCxrPreReceiveEarlyExits(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)

	tests := []struct {
		name    string
		prepare func()
		in      types.GithookPreReceiveInput
		wantErr string
		wantOut string
	}{
		{
			name:    "manage repo is rejected",
			in:      e.preInput(enum.GitOpTypeManageRepo),
			wantOut: "Push not allowed for repository management operations",
		},
		{
			name:    "merge queue ops pass",
			in:      e.preInput(enum.GitOpTypeMergeQueue, cxrUpdate(cxrRefMain)),
			wantOut: "",
		},
		{
			name:    "api refs only ops pass",
			in:      e.preInput(enum.GitOpTypeAPIRefsOnly, cxrCreate(cxrRefFeature)),
			wantOut: "",
		},
		{
			name: "unknown repository",
			in: types.GithookPreReceiveInput{
				GithookInputBase: types.GithookInputBase{RepoID: 9999, OperationType: enum.GitOpTypeGitPush},
			},
			wantErr: "failed to find repo",
		},
		{
			name: "invalid repository id",
			in: types.GithookPreReceiveInput{
				GithookInputBase: types.GithookInputBase{RepoID: 0, OperationType: enum.GitOpTypeGitPush},
			},
			wantErr: "valid repository reference",
		},
		{
			name:    "default branch cannot be deleted",
			in:      e.preInput(enum.GitOpTypeGitPush, cxrDelete(cxrRefMain)),
			wantOut: "default branch",
		},
		{
			name:    "pull request refs cannot be modified",
			in:      e.preInput(enum.GitOpTypeGitPush, cxrCreate("refs/pullreq/1/head")),
			wantOut: "pull request",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), tt.in)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantOut == "" {
				require.Nil(t, out.Error)
				return
			}
			require.NotNil(t, out.Error)
			require.Contains(t, *out.Error, tt.wantOut)
		})
	}
}

func TestCxrPreReceiveLinkedRepo(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	e.setRepoColumn(ctx, t, "repo_type", string(enum.RepoTypeLinked))

	out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), e.preInput(enum.GitOpTypeGitPush, cxrUpdate(cxrRefMain)))
	require.NoError(t, err)
	require.NotNil(t, out.Error)
	require.Contains(t, *out.Error, "linked repository")

	out, err = e.ctrl.PreReceive(ctx, e.git, e.session(),
		e.preInput(enum.GitOpTypeAPILinkedSync, cxrUpdate(cxrRefMain)))
	require.NoError(t, err)
	require.Nil(t, out.Error)
}

func TestCxrPreReceiveRepoState(t *testing.T) {
	ctx := t.Context()

	t.Run("git push is blocked in import state", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setRepoColumn(ctx, t, "repo_state", int(enum.RepoStateGitImport))
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(),
			e.preInput(enum.GitOpTypeGitPush, cxrUpdate(cxrRefFeature)))
		require.NoError(t, err)
		require.NotNil(t, out.Error)
		require.Contains(t, *out.Error, "Push not allowed when repository is in")
	})

	t.Run("migrate git push state skips rule evaluation and allows pull request refs", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setRepoColumn(ctx, t, "repo_state", int(enum.RepoStateMigrateGitPush))
		e.rules.rules = []types.RuleInfoInternal{cxrRule("branch", `{"lifecycle":{"create_forbidden":true}}`)}
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(),
			e.preInput(enum.GitOpTypeGitPush, cxrCreate("refs/pullreq/1/head"), cxrCreate(cxrRefFeature)))
		require.NoError(t, err)
		require.Nil(t, out.Error)
	})
}

func TestCxrPreReceiveFailures(t *testing.T) {
	ctx := t.Context()
	push := func(e *cxrEnv) types.GithookPreReceiveInput {
		return e.preInput(enum.GitOpTypeGitPush, cxrUpdate(cxrRefFeature))
	}

	t.Run("ancestor check fails", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.git.ancestorErr = errCxrBoom
		_, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.ErrorIs(t, err, errCxrBoom)
		require.ErrorContains(t, err, "failed to check branch ancestor")
	})

	t.Run("extender fails", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.pre.err = errCxrBoom
		_, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.ErrorContains(t, err, "failed to extend pre-receive hook")
	})

	t.Run("extender rejects the push", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		msg := "denied by extension"
		e.pre.errMsg = &msg
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.Equal(t, &msg, out.Error)
	})

	t.Run("rule listing fails", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.err = errCxrBoom
		_, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.ErrorContains(t, err, "failed to fetch protection rules")
	})

	t.Run("principal is unknown", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		in := push(e)
		in.PrincipalID = 9999
		_, err := e.ctrl.PreReceive(ctx, e.git, e.session(), in)
		require.ErrorContains(t, err, "failed to find principal")
	})

	t.Run("owner check fails", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.ctrl.authorizer = cxrAuthorizer{mode: cxrError}
		_, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.ErrorContains(t, err, "failed to determine if user is repo owner")
	})
}

func TestCxrPreReceiveRefRules(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name    string
		rule    types.RuleInfoInternal
		updates []hook.ReferenceUpdate
		blocked bool
		message string
	}{
		{
			name:    "branch creation forbidden",
			rule:    cxrRule("branch", `{"lifecycle":{"create_forbidden":true}}`),
			updates: []hook.ReferenceUpdate{cxrCreate(cxrRefFeature)},
			blocked: true,
			message: "Rule \"rule\" violation",
		},
		{
			name:    "branch deletion forbidden",
			rule:    cxrRule("branch", `{"lifecycle":{"delete_forbidden":true}}`),
			updates: []hook.ReferenceUpdate{cxrDelete(cxrRefFeature)},
			blocked: true,
		},
		{
			name:    "branch update forbidden",
			rule:    cxrRule("branch", `{"lifecycle":{"update_forbidden":true}}`),
			updates: []hook.ReferenceUpdate{cxrUpdate(cxrRefFeature)},
			blocked: true,
		},
		{
			name:    "tag creation forbidden",
			rule:    cxrRule("tag", `{"lifecycle":{"create_forbidden":true}}`),
			updates: []hook.ReferenceUpdate{cxrCreate(cxrRefTag)},
			blocked: true,
		},
		{
			name:    "tag deletion forbidden",
			rule:    cxrRule("tag", `{"lifecycle":{"delete_forbidden":true}}`),
			updates: []hook.ReferenceUpdate{cxrDelete(cxrRefTag)},
			blocked: true,
		},
		{
			name:    "tag force update forbidden",
			rule:    cxrRule("tag", `{"lifecycle":{"update_force_forbidden":true}}`),
			updates: []hook.ReferenceUpdate{cxrUpdate(cxrRefTag)},
			blocked: true,
		},
		{
			name:    "unrelated action is allowed",
			rule:    cxrRule("branch", `{"lifecycle":{"delete_forbidden":true}}`),
			updates: []hook.ReferenceUpdate{cxrCreate(cxrRefFeature)},
		},
		{
			name: "bypassed rule only warns",
			rule: cxrRule("branch",
				`{"bypass":{"repo_owners":true},"lifecycle":{"create_forbidden":true}}`),
			updates: []hook.ReferenceUpdate{cxrCreate(cxrRefFeature)},
			message: "Bypassed rule",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := cxrNewEnv(ctx, t)
			e.rules.rules = []types.RuleInfoInternal{tt.rule}
			out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), e.preInput(enum.GitOpTypeGitPush, tt.updates...))
			require.NoError(t, err)
			if tt.blocked {
				require.NotNil(t, out.Error)
				require.Contains(t, *out.Error, "blocked by protection rules")
			} else {
				require.Nil(t, out.Error)
			}
			if tt.message != "" {
				require.Contains(t, out.Messages[0], tt.message)
			}
		})
	}

	t.Run("API operations skip ref rules", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.rules = []types.RuleInfoInternal{cxrRule("branch", `{"lifecycle":{"create_forbidden":true}}`)}
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(),
			e.preInput(enum.GitOpTypeAPIContent, cxrCreate(cxrRefFeature)))
		require.NoError(t, err)
		require.Nil(t, out.Error)
	})

	t.Run("a rule with invalid definition fails the push", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.rules = []types.RuleInfoInternal{cxrRule("branch", "not json")}
		_, err := e.ctrl.PreReceive(ctx, e.git, e.session(),
			e.preInput(enum.GitOpTypeGitPush, cxrCreate(cxrRefFeature)))
		require.ErrorContains(t, err, "failed to check protection rules")
	})
}

func TestCxrPreReceiveMergeQueueProtection(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	e.mqQueued.queued["feature"] = struct{}{}

	out, err := e.ctrl.PreReceive(ctx, e.git, e.session(),
		e.preInput(enum.GitOpTypeGitPush, cxrUpdate(cxrRefFeature)))
	require.NoError(t, err)
	require.NotNil(t, out.Error)
	require.Contains(t, *out.Error, "blocked by protection rules")

	// Branches that are not queued and branch creations are fine.
	out, err = e.ctrl.PreReceive(ctx, e.git, e.session(),
		e.preInput(enum.GitOpTypeGitPush, cxrUpdate("refs/heads/other"), cxrCreate(cxrRefFeature)))
	require.NoError(t, err)
	require.Nil(t, out.Error)

	e.mqQueued.err = errCxrBoom
	_, err = e.ctrl.PreReceive(ctx, e.git, e.session(),
		e.preInput(enum.GitOpTypeGitPush, cxrDelete(cxrRefFeature)))
	require.ErrorIs(t, err, errCxrBoom)
}

func TestCxrPreReceivePushSettings(t *testing.T) {
	ctx := t.Context()
	push := func(e *cxrEnv) types.GithookPreReceiveInput {
		return e.preInput(enum.GitOpTypeGitPush, cxrCreate(cxrRefFeature))
	}

	t.Run("secret scanning blocks pushes with findings", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setSetting(ctx, t, settings.KeySecretScanningEnabled, true)
		e.git.scanFindings = []git.ScanSecretsFinding{{File: "a.txt", Description: "key", StartLine: 1}}
		e.git.branch = &git.GetBranchOutput{Branch: git.Branch{Name: "main", SHA: cxrSHA3}}

		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.NotNil(t, out.Error)
		require.Contains(t, *out.Error, "blocked by repository settings")
		require.Len(t, e.git.scanCalls, 1)
		require.Equal(t, cxrSHA3.String(), e.git.scanCalls[0].BaseRev)
	})

	t.Run("secret scanning without findings passes", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setSetting(ctx, t, settings.KeySecretScanningEnabled, true)
		e.git.branchErr = nil
		e.git.branch = &git.GetBranchOutput{Branch: git.Branch{Name: "main", SHA: cxrSHA3}}
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.Nil(t, out.Error)
	})

	t.Run("secret scanning failure fails the push", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setSetting(ctx, t, settings.KeySecretScanningEnabled, true)
		e.git.branch = &git.GetBranchOutput{Branch: git.Branch{Name: "main", SHA: cxrSHA3}}
		e.git.scanErr = errCxrBoom
		_, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.ErrorContains(t, err, "failed to scan secrets")
	})

	t.Run("oversize files block the push", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setSetting(ctx, t, settings.KeyFileSizeLimit, int64(100))
		e.git.processOut = git.ProcessPreReceiveObjectsOutput{
			FindOversizeFilesOutput: &git.FindOversizeFilesOutput{
				TotalsPerLimit:    map[int64]int64{100: 1},
				FileInfosPerLimit: map[int64][]git.FileInfo{100: {{SHA: cxrSHA1, Size: 500}}},
			},
		}
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.NotNil(t, out.Error)
		require.Contains(t, *out.Error, "blocked by repository settings")
		require.NotNil(t, e.git.processParams[0].FindOversizeFilesParams)
		require.EqualValues(t, 100, e.git.processParams[0].FindOversizeFilesParams.SizeLimit)
	})

	t.Run("committer mismatch blocks git pushes", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setSetting(ctx, t, settings.KeyPrincipalCommitterMatch, true)
		e.git.processOut = git.ProcessPreReceiveObjectsOutput{
			FindCommitterMismatchOutput: &git.FindCommitterMismatchOutput{
				CommitInfos: []git.CommitInfo{{SHA: cxrSHA1, Committer: "other@example.com"}}, Total: 1,
			},
		}
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.NotNil(t, out.Error)
		require.Equal(t, e.admin.Email, e.git.processParams[0].FindCommitterMismatchParams.PrincipalEmail)

		// API commits don't check committers.
		e.git.processParams = nil
		_, err = e.ctrl.PreReceive(ctx, e.git, e.session(),
			e.preInput(enum.GitOpTypeAPIContentBypassRules, cxrCreate(cxrRefFeature)))
		require.NoError(t, err)
		require.Nil(t, e.git.processParams[0].FindCommitterMismatchParams)
	})

	t.Run("unknown lfs objects block the push", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setSetting(ctx, t, settings.KeyGitLFSEnabled, true)
		e.git.processOut = git.ProcessPreReceiveObjectsOutput{
			FindLFSPointersOutput: &git.FindLFSPointersOutput{
				LFSInfos: []git.LFSInfo{{ObjID: "obj1", SHA: cxrSHA1}, {ObjID: "obj2", SHA: cxrSHA2}}, Total: 2,
			},
		}
		e.repoLFS.found = []*types.LFSObject{{OID: "obj1"}}
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.NotNil(t, out.Error)
		require.Contains(t, *out.Error, "blocked by repository settings")

		// All objects are known: the push passes.
		e.repoLFS.found = []*types.LFSObject{{OID: "obj1"}, {OID: "obj2"}}
		out, err = e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.Nil(t, out.Error)

		e.repoLFS.err = errCxrBoom
		_, err = e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.ErrorContains(t, err, "failed to find lfs objects")
	})

	t.Run("object processing failure fails the push", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setSetting(ctx, t, settings.KeyGitLFSEnabled, true)
		e.git.processErr = errCxrBoom
		_, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.ErrorContains(t, err, "failed to process pre-receive objects")
	})

	t.Run("deleting only branches skips object processing", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setSetting(ctx, t, settings.KeyGitLFSEnabled, true)
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), e.preInput(enum.GitOpTypeGitPush, cxrDelete(cxrRefFeature)))
		require.NoError(t, err)
		require.Nil(t, out.Error)
		require.Empty(t, e.git.processParams)
	})
}

func TestCxrPreReceivePushRules(t *testing.T) {
	ctx := t.Context()
	push := func(e *cxrEnv) types.GithookPreReceiveInput {
		return e.preInput(enum.GitOpTypeGitPush, cxrCreate(cxrRefFeature))
	}
	mismatch := &git.FindCommitterMismatchOutput{
		CommitInfos: []git.CommitInfo{{SHA: cxrSHA1, Committer: "other@example.com"}}, Total: 1,
	}

	t.Run("push rule with all checks blocks and propagates violations", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.rules = []types.RuleInfoInternal{cxrRule("push",
			`{"push":{"file_size_limit":100,"principal_committer_match":true,"secret_scanning_enabled":true}}`)}
		e.git.branch = &git.GetBranchOutput{Branch: git.Branch{Name: "main", SHA: cxrSHA3}}
		e.git.scanFindings = []git.ScanSecretsFinding{{File: "a.txt", Description: "key", StartLine: 1}}
		e.git.processOut = git.ProcessPreReceiveObjectsOutput{
			FindOversizeFilesOutput: &git.FindOversizeFilesOutput{
				TotalsPerLimit:    map[int64]int64{100: 1},
				FileInfosPerLimit: map[int64][]git.FileInfo{100: {{SHA: cxrSHA1, Size: 500}}},
			},
			FindCommitterMismatchOutput: mismatch,
		}
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.NotNil(t, out.Error)
		require.Contains(t, *out.Error, "blocked by protection rules")

		var violations []types.RuleViolations
		require.NoError(t, json.Unmarshal(out.RuleViolations, &violations))
		require.NotEmpty(t, violations)
		require.Empty(t, e.audit.actions)
	})

	t.Run("bypassed push rule is audited", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.rules = []types.RuleInfoInternal{cxrRule("push",
			`{"bypass":{"repo_owners":true},"push":{"principal_committer_match":true}}`)}
		e.git.processOut = git.ProcessPreReceiveObjectsOutput{FindCommitterMismatchOutput: mismatch}
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.Nil(t, out.Error)
		require.Equal(t, []audit.Action{audit.ActionBypassed}, e.audit.actions)
		require.NotEmpty(t, out.RuleViolations)

		// Failing audit must not fail the push.
		e.audit.err = errCxrBoom
		out, err = e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.Nil(t, out.Error)
	})

	t.Run("API commits without bypass intent are blocked", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.rules = []types.RuleInfoInternal{cxrRule("push",
			`{"bypass":{"repo_owners":true},"push":{"file_size_limit":100}}`)}
		oversize := &git.FindOversizeFilesOutput{
			TotalsPerLimit:    map[int64]int64{100: 1},
			FileInfosPerLimit: map[int64][]git.FileInfo{100: {{SHA: cxrSHA1, Size: 500}}},
		}
		e.git.processOut = git.ProcessPreReceiveObjectsOutput{FindOversizeFilesOutput: oversize}

		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(),
			e.preInput(enum.GitOpTypeAPIContent, cxrCreate(cxrRefFeature)))
		require.NoError(t, err)
		require.NotNil(t, out.Error)

		out, err = e.ctrl.PreReceive(ctx, e.git, e.session(),
			e.preInput(enum.GitOpTypeAPIContentBypassRules, cxrCreate(cxrRefFeature)))
		require.NoError(t, err)
		require.Nil(t, out.Error)
	})

	t.Run("push rule without checks needs no object processing", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.rules = []types.RuleInfoInternal{cxrRule("push", `{"push":{}}`)}
		out, err := e.ctrl.PreReceive(ctx, e.git, e.session(), push(e))
		require.NoError(t, err)
		require.Nil(t, out.Error)
	})
}

func TestCxrProcessProtectionViolations(t *testing.T) {
	tests := []struct {
		name     string
		settings *repoSettingsViolations
		wantErr  string
		wantMsgs []string
	}{
		{name: "nothing"},
		{
			name:     "secrets",
			settings: &repoSettingsViolations{SecretsFound: true},
			wantErr:  "blocked by repository settings",
			wantMsgs: []string{"Push blocked by repository settings: Secrets detected."},
		},
		{
			name:     "file size",
			settings: &repoSettingsViolations{ExceededFileSizeLimit: 42},
			wantErr:  "blocked by repository settings",
			wantMsgs: []string{"File size limit of 42 bytes exceeded."},
		},
		{
			name:     "committer",
			settings: &repoSettingsViolations{CommitterMismatchFound: true},
			wantErr:  "blocked by repository settings",
			wantMsgs: []string{"Committer user mismatch detected."},
		},
		{
			name:     "lfs",
			settings: &repoSettingsViolations{UnknownLFSObjectsFound: true},
			wantErr:  "blocked by repository settings",
			wantMsgs: []string{"Unknown Git LFS objects detected."},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &hook.Output{}
			processProtectionViolations(out, nil, tt.settings)
			if tt.wantErr == "" {
				require.Nil(t, out.Error)
				return
			}
			require.Equal(t, tt.wantErr, *out.Error)
			for _, m := range tt.wantMsgs {
				require.Contains(t, out.Messages[0], m)
			}
		})
	}

	t.Run("rule and settings violations are combined", func(t *testing.T) {
		out := &hook.Output{}
		critical := types.RuleViolations{
			Rule:       types.RuleInfo{Identifier: "r1", State: enum.RuleStateActive},
			Violations: []types.Violation{{Code: "c", Message: "bad"}},
		}
		processProtectionViolations(out, []types.RuleViolations{critical}, &repoSettingsViolations{SecretsFound: true})
		require.Equal(t, "blocked by protection rules, blocked by repository settings", *out.Error)
		require.Len(t, out.Messages, 2)
	})
}

func TestCxrGetBaseSHAForScanningChanges(t *testing.T) {
	ctx := t.Context()
	repo := &types.RepositoryCore{ID: 1, GitUID: "git", DefaultBranch: "main"}
	branchOut := &git.GetBranchOutput{Branch: git.Branch{Name: "main", SHA: cxrSHA3}}

	tests := []struct {
		name    string
		git     *cxrGit
		updates []hook.ReferenceUpdate
		target  hook.ReferenceUpdate
		wantSHA string
		wantOK  bool
		wantErr bool
	}{
		{
			name:    "existing ref uses its old sha",
			git:     &cxrGit{},
			target:  cxrUpdate(cxrRefFeature),
			wantSHA: cxrSHA1.String(), wantOK: true,
		},
		{
			name:    "default branch updated in the same push",
			git:     &cxrGit{},
			updates: []hook.ReferenceUpdate{cxrUpdate(cxrRefMain), cxrCreate(cxrRefFeature)},
			target:  cxrCreate(cxrRefFeature),
			wantSHA: cxrSHA1.String(), wantOK: true,
		},
		{
			name:    "default branch created in the same push",
			git:     &cxrGit{},
			updates: []hook.ReferenceUpdate{cxrCreate(cxrRefMain), cxrCreate(cxrRefFeature)},
			target:  cxrCreate(cxrRefFeature),
		},
		{
			name:    "default branch read from git",
			git:     &cxrGit{branch: branchOut},
			updates: []hook.ReferenceUpdate{cxrCreate(cxrRefFeature)},
			target:  cxrCreate(cxrRefFeature),
			wantSHA: cxrSHA3.String(), wantOK: true,
		},
		{
			name:    "empty repository has no base",
			git:     &cxrGit{branchErr: errors.NotFound("no branch")},
			updates: []hook.ReferenceUpdate{cxrCreate(cxrRefFeature)},
			target:  cxrCreate(cxrRefFeature),
		},
		{
			name:    "git failure",
			git:     &cxrGit{branchErr: errCxrBoom},
			updates: []hook.ReferenceUpdate{cxrCreate(cxrRefFeature)},
			target:  cxrCreate(cxrRefFeature),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := GetBaseSHAForScanningChanges(
				ctx, tt.git, repo, hook.Environment{}, tt.updates, tt.target)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.Equal(t, tt.wantSHA, got.String())
			}
		})
	}
}
