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
	"context"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth/authz/authztest"
	"github.com/harness/gitness/app/services/autolink"
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/git/api"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type crpAccessCall func(c *Controller, ref string) error

type crpAccessCase struct {
	name string
	// call invokes the controller method with valid-looking input.
	call crpAccessCall
	// gitStates is true for methods that use getRepoCheckAccessForGit (importing states are allowed).
	gitStates bool
}

func crpDiscard[T any](_ T, err error) error { return err }

func crpDiscard2[T, U any](_ T, _ U, err error) error { return err }

func crpAccessCases() []crpAccessCase {
	ctx := context.Background()
	s := crpSession()

	cases := []crpAccessCase{
		{name: "Archive", call: func(c *Controller, ref string) error {
			return c.Archive(ctx, s, ref, api.ArchiveParams{}, &bytes.Buffer{})
		}},
		{name: "Blame", call: func(c *Controller, ref string) error {
			return crpDiscard(c.Blame(ctx, s, ref, "", "file.txt", 0, 0))
		}},
		{name: "CodeOwnersRaw", call: func(c *Controller, ref string) error {
			return crpDiscard(c.CodeOwnersRaw(ctx, s, ref, ""))
		}},
		{name: "CodeOwnersValidate", call: func(c *Controller, ref string) error {
			return crpDiscard(c.CodeOwnersValidate(ctx, s, ref, ""))
		}},
		{name: "GetContent", call: func(c *Controller, ref string) error {
			return crpDiscard(c.GetContent(ctx, s, ref, "", "", false, false))
		}},
		{name: "PathsDetails", call: func(c *Controller, ref string) error {
			return crpDiscard(c.PathsDetails(ctx, s, ref, "", PathsDetailsInput{}))
		}},
		{name: "CommitDiff", call: func(c *Controller, ref string) error {
			return c.CommitDiff(ctx, s, ref, "rev", false, &bytes.Buffer{})
		}},
		{name: "RawDiff", call: func(c *Controller, ref string) error {
			return c.RawDiff(ctx, &bytes.Buffer{}, s, ref, "a..b", false)
		}},
		{name: "GetBranch", call: func(c *Controller, ref string) error {
			return crpDiscard(c.GetBranch(ctx, s, ref, "b", types.BranchMetadataOptions{}))
		}},
		{name: "GetCommit", call: func(c *Controller, ref string) error {
			return crpDiscard(c.GetCommit(ctx, s, ref, "rev"))
		}},
		{name: "GetCommitDivergences", call: func(c *Controller, ref string) error {
			return crpDiscard(c.GetCommitDivergences(ctx, s, ref, nil))
		}},
		{name: "ListBranches", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.ListBranches(ctx, s, ref, &types.BranchFilter{}))
		}},
		{name: "ListCommitTags", call: func(c *Controller, ref string) error {
			return crpDiscard(c.ListCommitTags(ctx, s, ref, false, &types.TagFilter{}))
		}},
		{name: "ListCommits", call: func(c *Controller, ref string) error {
			return crpDiscard(c.ListCommits(ctx, s, ref, "", &types.CommitFilter{}))
		}},
		{name: "ListPaths", call: func(c *Controller, ref string) error {
			return crpDiscard(c.ListPaths(ctx, s, ref, "", false))
		}},
		{name: "MergeCheck", call: func(c *Controller, ref string) error {
			return crpDiscard(c.MergeCheck(ctx, s, ref, "a..b"))
		}},
		{name: "Raw", call: func(c *Controller, ref string) error {
			return crpDiscard(c.Raw(ctx, s, ref, "", "file.txt"))
		}},
		{name: "Summary", call: func(c *Controller, ref string) error {
			return crpDiscard(c.Summary(ctx, s, ref))
		}},
		{name: "ListActivities", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.ListActivities(ctx, s, ref, &types.RepoActivityFilter{}))
		}},
		{name: "RuleList", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.RuleList(ctx, s, ref, false, &types.RuleFilter{}))
		}},
		{name: "RuleFind", call: func(c *Controller, ref string) error {
			return crpDiscard(c.RuleFind(ctx, s, ref, "rule"))
		}},
		{name: "ListLabels", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.ListLabels(ctx, s, ref, &types.LabelFilter{}))
		}},
		{name: "FindLabel", call: func(c *Controller, ref string) error {
			return crpDiscard(c.FindLabel(ctx, s, ref, "key", false))
		}},
		{name: "ListLabelValues", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.ListLabelValues(ctx, s, ref, "key", types.ListQueryFilter{}))
		}},
		{name: "AutolinkList", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.AutolinkList(ctx, s, ref, &types.AutoLinkFilter{}))
		}},
		{name: "AutolinkFind", call: func(c *Controller, ref string) error {
			return crpDiscard(c.AutolinkFind(ctx, s, ref, 1))
		}},
		{name: "MergeQueueListEntries", call: func(c *Controller, ref string) error {
			return crpDiscard(c.MergeQueueListEntries(ctx, s, ref, crpDefaultBranch))
		}},
		// push permission.
		{name: "CreateBranch", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.CreateBranch(ctx, s, ref, &CreateBranchInput{Name: "b"}))
		}},
		{name: "DeleteBranch", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.DeleteBranch(ctx, s, ref, "b", false, false))
		}},
		{name: "CreateCommitTag", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.CreateCommitTag(ctx, s, ref, &CreateCommitTagInput{Name: "t"}))
		}},
		{name: "DeleteCommitTag", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.DeleteCommitTag(ctx, s, ref, "t", false, false))
		}},
		{name: "CommitFiles", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.CommitFiles(ctx, s, ref, &CommitFilesOptions{}))
		}},
		{name: "PipelineGenerate", call: func(c *Controller, ref string) error {
			return crpDiscard(c.PipelineGenerate(ctx, s, ref))
		}},
		{name: "ForkSync", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.ForkSync(ctx, s, ref, &ForkSyncInput{}))
		}},
		{name: "MergeCommit", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.MergeCommit(ctx, s, ref, &MergeCommitInput{
				BaseBranch: crpDefaultBranch, HeadBranch: crpHeadBranch, HeadCommitSHA: crpTestSHA,
			}))
		}},
		{name: "Rebase", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.Rebase(ctx, s, ref, &RebaseInput{
				BaseBranch: crpDefaultBranch, HeadBranch: crpHeadBranch, HeadCommitSHA: crpTestSHA,
			}))
		}},
		{name: "Squash", call: func(c *Controller, ref string) error {
			return crpDiscard2(c.Squash(ctx, s, ref, &SquashInput{
				BaseBranch: crpDefaultBranch, HeadBranch: crpHeadBranch, HeadCommitSHA: crpTestSHA,
			}))
		}},
		// edit permission.
		{name: "UpdateDefaultBranch", call: func(c *Controller, ref string) error {
			return crpDiscard(c.UpdateDefaultBranch(ctx, s, ref, &UpdateDefaultBranchInput{Name: "dev"}))
		}},
		{name: "RuleCreate", call: func(c *Controller, ref string) error {
			return crpDiscard(c.RuleCreate(ctx, s, ref, &rules.CreateInput{}))
		}},
		{name: "RuleUpdate", call: func(c *Controller, ref string) error {
			return crpDiscard(c.RuleUpdate(ctx, s, ref, "rule", &rules.UpdateInput{}))
		}},
		{name: "RuleDelete", call: func(c *Controller, ref string) error {
			return c.RuleDelete(ctx, s, ref, "rule")
		}},
		{name: "DefineLabel", call: func(c *Controller, ref string) error {
			return crpDiscard(c.DefineLabel(ctx, s, ref, &types.DefineLabelInput{}))
		}},
		{name: "SaveLabel", call: func(c *Controller, ref string) error {
			return crpDiscard(c.SaveLabel(ctx, s, ref, &types.SaveInput{}))
		}},
		{name: "UpdateLabel", call: func(c *Controller, ref string) error {
			return crpDiscard(c.UpdateLabel(ctx, s, ref, "key", &types.UpdateLabelInput{}))
		}},
		{name: "DeleteLabel", call: func(c *Controller, ref string) error {
			return c.DeleteLabel(ctx, s, ref, "key")
		}},
		{name: "DefineLabelValue", call: func(c *Controller, ref string) error {
			return crpDiscard(c.DefineLabelValue(ctx, s, ref, "key", &types.DefineValueInput{}))
		}},
		{name: "UpdateLabelValue", call: func(c *Controller, ref string) error {
			return crpDiscard(c.UpdateLabelValue(ctx, s, ref, "key", "value", &types.UpdateValueInput{}))
		}},
		{name: "DeleteLabelValue", call: func(c *Controller, ref string) error {
			return c.DeleteLabelValue(ctx, s, ref, "key", "value")
		}},
		{name: "AutolinkCreate", call: func(c *Controller, ref string) error {
			return crpDiscard(c.AutolinkCreate(ctx, s, ref, &autolink.AutoLinkInput{}))
		}},
		{name: "AutolinkUpdate", call: func(c *Controller, ref string) error {
			return crpDiscard(c.AutolinkUpdate(ctx, s, ref, 1, &autolink.AutoLinkUpdateInput{}))
		}},
		{name: "AutolinkDelete", call: func(c *Controller, ref string) error {
			return c.AutolinkDelete(ctx, s, ref, 1)
		}},
		{name: "MergeQueueClear", call: func(c *Controller, ref string) error {
			return c.MergeQueueClear(ctx, s, ref, crpDefaultBranch)
		}},
		{name: "Move", call: func(c *Controller, ref string) error {
			return crpDiscard(c.Move(ctx, s, ref, &MoveInput{}))
		}},
		{name: "CreateFork", call: func(c *Controller, ref string) error {
			return crpDiscard(c.CreateFork(ctx, s, ref, &CreateForkInput{}))
		}},
		// git service methods allow importing states.
		{name: "GitInfoRefs", gitStates: true, call: func(c *Controller, ref string) error {
			return c.GitInfoRefs(ctx, s, ref, enum.GitServiceTypeUploadPack, "", &bytes.Buffer{})
		}},
		{name: "GitServicePackRead", gitStates: true, call: func(c *Controller, ref string) error {
			return c.GitServicePack(ctx, s, ref, api.ServicePackOptions{Service: enum.GitServiceTypeUploadPack})
		}},
		{name: "GitServicePackWrite", gitStates: true, call: func(c *Controller, ref string) error {
			return c.GitServicePack(ctx, s, ref, api.ServicePackOptions{Service: enum.GitServiceTypeReceivePack})
		}},
		{name: "GetLanguages", gitStates: true, call: func(c *Controller, ref string) error {
			return crpDiscard(c.GetLanguages(ctx, s, ref))
		}},
	}

	return cases
}

func TestCrpAccess_UnknownRepo(t *testing.T) {
	for _, tc := range crpAccessCases() {
		t.Run(tc.name, func(t *testing.T) {
			e := crpNewEnv(t)

			err := tc.call(e.c, crpUnknownRepoRef)

			require.Error(t, err)
			assert.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		})
	}
}

func TestCrpAccess_EmptyRepoRef(t *testing.T) {
	for _, tc := range crpAccessCases() {
		t.Run(tc.name, func(t *testing.T) {
			e := crpNewEnv(t)

			err := tc.call(e.c, "")

			require.Error(t, err)
			assert.ErrorContains(t, err, "valid repository reference")
		})
	}
}

func TestCrpAccess_Denied(t *testing.T) {
	for _, tc := range crpAccessCases() {
		t.Run(tc.name, func(t *testing.T) {
			e := crpNewEnv(t)
			e.setAuthorizer(authztest.DenyAuthorizer{})

			err := tc.call(e.c, crpRepoRef)

			require.Error(t, err)
			assert.ErrorIs(t, err, apiauth.ErrForbidden)
		})
	}
}

func TestCrpAccess_AuthorizerFailure(t *testing.T) {
	for _, tc := range crpAccessCases() {
		t.Run(tc.name, func(t *testing.T) {
			e := crpNewEnv(t)
			e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})

			err := tc.call(e.c, crpRepoRef)

			require.Error(t, err)
			assert.ErrorIs(t, err, errCrpBoom)
		})
	}
}

func TestCrpAccess_RepoStateRestriction(t *testing.T) {
	for _, tc := range crpAccessCases() {
		if tc.gitStates {
			continue
		}

		t.Run(tc.name, func(t *testing.T) {
			e := crpNewEnv(t)
			e.setRepo(func(r *types.RepositoryCore) { r.State = enum.RepoStateGitImport })

			err := tc.call(e.c, crpRepoRef)

			require.Error(t, err)
			assert.ErrorContains(t, err, "not allowed for repository in state")
		})
	}
}

func TestCrpAccess_LinkedRepoIsReadOnly(t *testing.T) {
	writeCases := map[string]bool{
		"CreateBranch": true, "DeleteBranch": true, "CreateCommitTag": true, "DeleteCommitTag": true,
		"CommitFiles": true, "UpdateDefaultBranch": true, "RuleCreate": true, "DefineLabel": true,
		"AutolinkCreate": true, "Move": true, "PipelineGenerate": true, "ForkSync": true,
		"MergeCommit": true, "Rebase": true, "Squash": true, "MergeQueueClear": true,
	}

	for _, tc := range crpAccessCases() {
		if !writeCases[tc.name] {
			continue
		}

		t.Run(tc.name, func(t *testing.T) {
			e := crpNewEnv(t)
			e.setRepo(func(r *types.RepositoryCore) { r.Type = enum.RepoTypeLinked })

			err := tc.call(e.c, crpRepoRef)

			require.Error(t, err)
			assert.ErrorContains(t, err, "linked repository")
		})
	}
}
