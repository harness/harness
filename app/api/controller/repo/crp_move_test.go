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
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func crpStrPtr(s string) *string { return &s }

func TestCrpMoveInput_HasChanges(t *testing.T) {
	repo := &types.Repository{Identifier: "repo"}
	parent := &types.SpaceCore{ID: 3}

	tests := []struct {
		name   string
		in     MoveInput
		target *types.SpaceCore
		want   bool
	}{
		{name: "nothing requested", in: MoveInput{}, target: parent},
		{name: "same identifier", in: MoveInput{Identifier: crpStrPtr("repo")}, target: parent},
		{name: "new identifier", in: MoveInput{Identifier: crpStrPtr(crpOtherSpacePath)}, target: parent, want: true},
		{name: "same parent", in: MoveInput{ParentRef: crpStrPtr("3")}, target: parent},
		{name: "other parent", in: MoveInput{ParentRef: crpStrPtr("4")}, target: &types.SpaceCore{ID: 4}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.in.hasChanges(repo, parent, tt.target))
		})
	}
}

func TestCrpMove_InputValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("identifier check", func(t *testing.T) {
		e := crpNewEnv(t)
		e.c.identifierCheck = func(string, *auth.Session) error { return errCrpBoom }

		_, err := e.c.Move(ctx, crpSession(), crpRepoRef, &MoveInput{Identifier: crpStrPtr("x")})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("invalid parent reference", func(t *testing.T) {
		e := crpNewEnv(t)

		_, err := e.c.Move(ctx, crpSession(), crpRepoRef, &MoveInput{ParentRef: crpStrPtr("  ")})

		require.ErrorIs(t, err, errRepositoryRequiresParent)
	})

	t.Run("legacy uid is used as identifier", func(t *testing.T) {
		e := crpNewEnv(t)
		in := &MoveInput{UID: crpStrPtr("legacy")}

		require.NoError(t, e.c.sanitizeMoveInput(in, crpSession()))

		require.NotNil(t, in.Identifier)
		assert.Equal(t, "legacy", *in.Identifier)
	})
}

func TestCrpMove(t *testing.T) {
	ctx := context.Background()

	t.Run("without changes the repository is returned as is", func(t *testing.T) {
		e := crpNewEnv(t)

		out, err := e.c.Move(ctx, crpSession(), crpRepoRef, &MoveInput{Identifier: crpStrPtr("repo")})

		require.NoError(t, err)
		assert.Equal(t, "repo", out.Identifier)
		assert.False(t, e.repoStore.called("UpdateOptLock"))
	})

	t.Run("renames the repository", func(t *testing.T) {
		e := crpNewEnv(t)
		e.public.public[crpRepoPath] = true

		out, err := e.c.Move(ctx, crpSession(), crpRepoRef, &MoveInput{Identifier: crpStrPtr("renamed")})

		require.NoError(t, err)
		assert.Equal(t, "renamed", out.Identifier)
		assert.Equal(t, "https://git.example.com/"+out.Path+".git", out.GitURL)
		assert.False(t, e.repoStore.called("UpdateRootSpace"), "root space is unchanged")
		assert.Equal(t, []string{crpRepoPath}, e.public.deleted)
		assert.True(t, e.public.public[out.Path], "public access follows the repository")
	})

	t.Run("moves the repository to another root space", func(t *testing.T) {
		e := crpNewEnv(t)
		e.addOtherSpace()
		e.c.pullReqStore = &crpPullReqStore{}

		out, err := e.c.Move(ctx, crpSession(), crpRepoRef, &MoveInput{ParentRef: crpStrPtr("4")})

		require.NoError(t, err)
		assert.EqualValues(t, 4, out.ParentID)
		assert.EqualValues(t, 4, out.RootSpaceID)
		assert.Equal(t, crpOtherSpacePath, out.RootSpaceIdentifier)
		assert.True(t, e.repoStore.called("UpdateRootSpace"))
	})

	failures := []struct {
		name  string
		setup func(e *crpEnv)
		in    *MoveInput
	}{
		{name: "no permission", in: &MoveInput{}, setup: func(e *crpEnv) {
			e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})
		}},
		{name: "repo lookup failure", in: &MoveInput{}, setup: func(e *crpEnv) {
			e.repoStore.errs["Find"] = errCrpBoom
		}},
		{name: "unknown current parent", in: &MoveInput{}, setup: func(e *crpEnv) {
			delete(e.spaceIDs.spaces, 3)
		}},
		{name: "no access to the target space", in: &MoveInput{ParentRef: crpStrPtr("4")}, setup: func(e *crpEnv) {
			e.addOtherSpace()
			e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoCreate: true}})
		}},
		{name: "target space lookup failure", in: &MoveInput{ParentRef: crpStrPtr("4")}, setup: func(e *crpEnv) {
			e.addOtherSpace()
			e.spaces.err = errCrpBoom
		}},
		{name: "public access lookup failure", in: &MoveInput{Identifier: crpStrPtr("x")}, setup: func(e *crpEnv) {
			e.public.getErr = errCrpBoom
		}},
		{name: "public access removal failure", in: &MoveInput{Identifier: crpStrPtr("x")}, setup: func(e *crpEnv) {
			e.public.deleteErr = errCrpBoom
		}},
		{name: "update failure", in: &MoveInput{Identifier: crpStrPtr("x")}, setup: func(e *crpEnv) {
			e.repoStore.errs["UpdateOptLock"] = errCrpBoom
		}},
		{name: "root space update failure", in: &MoveInput{ParentRef: crpStrPtr("4")}, setup: func(e *crpEnv) {
			e.addOtherSpace()
			e.repoStore.errs["UpdateRootSpace"] = errCrpBoom
		}},
		{name: "pull request root space update failure", in: &MoveInput{ParentRef: crpStrPtr("4")},
			setup: func(e *crpEnv) {
				e.addOtherSpace()
				e.c.pullReqStore = &crpPullReqStore{rootErr: errCrpBoom}
			}},
		{name: "public access set failure reverts the move", in: &MoveInput{Identifier: crpStrPtr("x")},
			setup: func(e *crpEnv) { e.public.setErr = errCrpBoom }},
		{name: "public access set failure and failed revert", in: &MoveInput{Identifier: crpStrPtr("x")},
			setup: func(e *crpEnv) {
				e.public.setErr = errCrpBoom
				e.public.deleteErr = errCrpBoom
			}},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			tt.setup(e)

			_, err := e.c.Move(ctx, crpSession(), crpRepoRef, tt.in)

			require.Error(t, err)
		})
	}

	t.Run("public access set failure restores the original repository", func(t *testing.T) {
		e := crpNewEnv(t)
		e.public.setErr = errCrpBoom

		_, err := e.c.Move(ctx, crpSession(), crpRepoRef, &MoveInput{Identifier: crpStrPtr("x")})

		require.ErrorIs(t, err, errCrpBoom)
		assert.Equal(t, "repo", e.repoStore.repos[crpRepoID].Identifier)
	})

	t.Run("failed update restores public access", func(t *testing.T) {
		e := crpNewEnv(t)
		e.public.public[crpRepoPath] = true
		e.repoStore.errs["UpdateOptLock"] = errCrpBoom

		_, err := e.c.Move(ctx, crpSession(), crpRepoRef, &MoveInput{Identifier: crpStrPtr("x")})

		require.ErrorIs(t, err, errCrpBoom)
		assert.True(t, e.public.public[crpRepoPath])
	})
}
