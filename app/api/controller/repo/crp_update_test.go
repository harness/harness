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
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func crpStatePtr(s enum.RepoState) *enum.RepoState { return &s }

func crpTagsPtr(t types.RepoTags) *types.RepoTags { return &t }

func TestCrpHasTagChanges(t *testing.T) {
	repo := &types.Repository{Tags: json.RawMessage(`{"a":"1","b":"2"}`)}

	tests := []struct {
		name string
		tags *types.RepoTags
		repo *types.Repository
		want bool
	}{
		{name: "no tags in input", tags: nil, repo: repo},
		{name: "same tags", tags: crpTagsPtr(types.RepoTags{"a": "1", "b": "2"}), repo: repo},
		{name: "different count", tags: crpTagsPtr(types.RepoTags{"a": "1"}), repo: repo, want: true},
		{name: "different value", tags: crpTagsPtr(types.RepoTags{"a": "1", "b": "3"}), repo: repo, want: true},
		{name: "different key", tags: crpTagsPtr(types.RepoTags{"a": "1", "c": "2"}), repo: repo, want: true},
		{name: "clearing tags", tags: crpTagsPtr(types.RepoTags{}), repo: repo, want: true},
		{
			name: "tags on a repo without tags", tags: crpTagsPtr(types.RepoTags{"a": "1"}),
			repo: &types.Repository{}, want: true,
		},
		{name: "empty tags on a repo without tags", tags: crpTagsPtr(types.RepoTags{}), repo: &types.Repository{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasTagChanges(&UpdateInput{Tags: tt.tags}, tt.repo))
		})
	}
}

func TestCrpUpdateInput_HasChanges(t *testing.T) {
	desc := "desc"
	other := crpOtherSpacePath
	repo := &types.Repository{Description: desc, State: enum.RepoStateActive}

	tests := []struct {
		name string
		in   UpdateInput
		want bool
	}{
		{name: "empty", in: UpdateInput{}},
		{name: "same description", in: UpdateInput{Description: &desc}},
		{name: "new description", in: UpdateInput{Description: &other}, want: true},
		{name: "same state", in: UpdateInput{State: crpStatePtr(enum.RepoStateActive)}},
		{name: "new state", in: UpdateInput{State: crpStatePtr(enum.RepoStateArchived)}, want: true},
		{name: "new tags", in: UpdateInput{Tags: crpTagsPtr(types.RepoTags{"k": "v"})}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.in.hasChanges(repo))
		})
	}
}

func TestCrpUpdate(t *testing.T) {
	ctx := context.Background()
	desc := "  a new description  "

	t.Run("updates the description", func(t *testing.T) {
		e := crpNewEnv(t)

		out, err := e.c.Update(ctx, crpSession(), crpRepoRef, &UpdateInput{Description: &desc})

		require.NoError(t, err)
		assert.Equal(t, "a new description", out.Description)
		assert.Equal(t, "https://git.example.com/space/repo.git", out.GitURL)
		assert.Equal(t, "a new description", e.repoStore.repos[crpRepoID].Description)
	})

	t.Run("updates the tags", func(t *testing.T) {
		e := crpNewEnv(t)
		tags := types.RepoTags{"team": "core"}

		_, err := e.c.Update(ctx, crpSession(), crpRepoRef, &UpdateInput{Tags: &tags})

		require.NoError(t, err)
		assert.JSONEq(t, `{"team":"core"}`, string(e.repoStore.repos[crpRepoID].Tags))
	})

	t.Run("archives and reactivates", func(t *testing.T) {
		e := crpNewEnv(t)

		out, err := e.c.Update(ctx, crpSession(), crpRepoRef, &UpdateInput{State: crpStatePtr(enum.RepoStateArchived)})
		require.NoError(t, err)
		assert.True(t, out.Archived)

		e.setRepo(func(r *types.RepositoryCore) { r.State = enum.RepoStateArchived })
		out, err = e.c.Update(ctx, crpSession(), crpRepoRef, &UpdateInput{State: crpStatePtr(enum.RepoStateActive)})
		require.NoError(t, err)
		assert.False(t, out.Archived)
	})

	t.Run("no changes returns the repository without writing", func(t *testing.T) {
		e := crpNewEnv(t)
		same := ""

		out, err := e.c.Update(ctx, crpSession(), crpRepoRef, &UpdateInput{Description: &same})

		require.NoError(t, err)
		assert.Equal(t, crpRepoID, out.ID)
		assert.False(t, e.repoStore.called("UpdateOptLock"))
	})

	failures := []struct {
		name  string
		setup func(e *crpEnv)
		ref   string
		in    *UpdateInput
		check func(t *testing.T, err error)
	}{
		{name: "empty ref", ref: "", in: &UpdateInput{}, check: func(t *testing.T, err error) {
			require.ErrorContains(t, err, "valid repository reference")
		}},
		{name: "archived repos can't be edited", ref: crpRepoRef, in: &UpdateInput{Description: &desc},
			setup: func(e *crpEnv) { e.setRepo(func(r *types.RepositoryCore) { r.State = enum.RepoStateArchived }) },
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, "not allowed for repository in state") }},
		{name: "no permission", ref: crpRepoRef, in: &UpdateInput{Description: &desc},
			setup: func(e *crpEnv) { e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom}) },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "store lookup failure", ref: crpRepoRef, in: &UpdateInput{Description: &desc},
			setup: func(e *crpEnv) { e.repoStore.errs["Find"] = errCrpBoom },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "description too long", ref: crpRepoRef, in: &UpdateInput{Description: crpLongString(1025)},
			check: func(t *testing.T, err error) { require.Error(t, err) }},
		{name: "invalid tags", ref: crpRepoRef,
			in:    &UpdateInput{Tags: crpTagsPtr(types.RepoTags{"a:b": "c"})},
			check: func(t *testing.T, err error) { require.Error(t, err) }},
		{name: "lifecycle restriction", ref: crpRepoRef, in: &UpdateInput{Description: &desc},
			setup: func(e *crpEnv) { e.c.repoCheck = crpRepoCheck{lifecycleErr: errCrpBoom} },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "state transition not allowed", ref: crpRepoRef,
			in: &UpdateInput{State: crpStatePtr(enum.RepoStateGitImport)},
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
			}},
		{name: "update failure", ref: crpRepoRef, in: &UpdateInput{Description: &desc},
			setup: func(e *crpEnv) { e.repoStore.errs["UpdateOptLock"] = errCrpBoom },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "output failure", ref: crpRepoRef, in: &UpdateInput{Description: &desc},
			setup: func(e *crpEnv) { e.public.getErr = errCrpBoom },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			if tt.setup != nil {
				tt.setup(e)
			}

			_, err := e.c.Update(ctx, crpSession(), tt.ref, tt.in)

			tt.check(t, err)
		})
	}
}

func crpLongString(n int) *string {
	s := strings.Repeat("a", n)

	return &s
}

func TestCrpUpdatePublicAccess(t *testing.T) {
	ctx := context.Background()
	in := func(public bool) *UpdatePublicAccessInput { return &UpdatePublicAccessInput{IsPublic: public} }

	t.Run("makes the repository public and private again", func(t *testing.T) {
		e := crpNewEnv(t)

		out, err := e.c.UpdatePublicAccess(ctx, crpSession(), crpRepoRef, in(true))
		require.NoError(t, err)
		assert.True(t, out.IsPublic)
		assert.True(t, e.public.public[crpRepoPath])

		out, err = e.c.UpdatePublicAccess(ctx, crpSession(), crpRepoRef, in(false))
		require.NoError(t, err)
		assert.False(t, out.IsPublic)
		assert.False(t, e.public.public[crpRepoPath])
	})

	t.Run("no-op when the access already matches", func(t *testing.T) {
		e := crpNewEnv(t)
		e.public.public[crpRepoPath] = true

		out, err := e.c.UpdatePublicAccess(ctx, crpSession(), crpRepoRef, in(true))

		require.NoError(t, err)
		assert.True(t, out.IsPublic)
		assert.Empty(t, e.public.deleted)
	})

	failures := []struct {
		name  string
		setup func(e *crpEnv)
		in    *UpdatePublicAccessInput
		check func(t *testing.T, err error)
	}{
		{name: "forks can't be public", in: in(true),
			setup: func(e *crpEnv) { e.setRepo(func(r *types.RepositoryCore) { r.ForkID = 55 }) },
			check: func(t *testing.T, err error) { assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err)) }},
		{name: "no permission", in: in(true),
			setup: func(e *crpEnv) { e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom}) },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "support lookup failure", in: in(true),
			setup: func(e *crpEnv) { e.public.supportedErr = errCrpBoom },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "public access unsupported", in: in(true),
			setup: func(e *crpEnv) { e.public.unsupported = true },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errPublicRepoCreationDisabled) }},
		{name: "current status lookup failure", in: in(true),
			setup: func(e *crpEnv) { e.public.getErr = errCrpBoom },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "store failure", in: in(true),
			setup: func(e *crpEnv) { e.repoStore.errs["Find"] = errCrpBoom },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "set failure", in: in(true),
			setup: func(e *crpEnv) { e.public.setErr = errCrpBoom },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "no-op output failure", in: in(false),
			setup: func(e *crpEnv) { e.repoStore.repos[crpRepoID].ForkID = 555 },
			check: func(t *testing.T, err error) { require.Error(t, err) }},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			tt.setup(e)

			_, err := e.c.UpdatePublicAccess(ctx, crpSession(), crpRepoRef, tt.in)

			tt.check(t, err)
		})
	}
}
