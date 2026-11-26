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
	"testing"

	apphook "github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestCxrUpdateHook(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)

	in := types.GithookUpdateInput{
		GithookInputBase: types.GithookInputBase{RepoID: e.repo.ID, PrincipalID: e.admin.ID},
		UpdateInput:      hook.UpdateInput{RefUpdate: cxrUpdate(cxrRefMain)},
	}
	out, err := e.ctrl.Update(ctx, e.git, e.session(), in)
	require.NoError(t, err)
	require.Equal(t, hook.Output{}, out)
	require.Equal(t, 1, e.update.calls)

	e.update.err = errCxrBoom
	_, err = e.ctrl.Update(ctx, e.git, e.session(), in)
	require.ErrorContains(t, err, "failed to extend update hook")

	in.RepoID = 0
	_, err = e.ctrl.Update(ctx, e.git, e.session(), in)
	require.ErrorContains(t, err, "valid repository reference")
}

func TestCxrNoOpExtenders(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	out := &hook.Output{}

	require.NoError(t, NewPreReceiveExtender().Extend(
		ctx, e.git, e.session(), e.repo.Core(), e.preInput(enum.GitOpTypeGitPush), out))
	require.NoError(t, NewUpdateExtender().Extend(
		ctx, e.git, e.session(), e.repo.Core(), types.GithookUpdateInput{}, out))
	require.NoError(t, NewPostReceiveExtender().Extend(
		ctx, e.git, e.session(), e.repo.Core(), e.postInput(enum.GitOpTypeGitPush), out))
	require.Equal(t, hook.Output{}, *out)
}

func TestCxrControllerClientFactory(t *testing.T) {
	factory := &ControllerClientFactory{githookCtrl: &Controller{}}

	t.Run("missing payload", func(t *testing.T) {
		_, err := factory.NewClient(map[string]string{})
		require.ErrorContains(t, err, "failed to load payload")
	})

	t.Run("disabled hooks return a noop client", func(t *testing.T) {
		env, err := hook.GenerateEnvironmentVariables(apphook.Payload{Disabled: true})
		require.NoError(t, err)
		client, err := factory.NewClient(env)
		require.NoError(t, err)
		out, err := client.PreReceive(t.Context(), hook.PreReceiveInput{})
		require.NoError(t, err)
		require.Equal(t, []string{"hook disabled"}, out.Messages)
	})

	t.Run("invalid payload", func(t *testing.T) {
		env, err := hook.GenerateEnvironmentVariables(apphook.Payload{})
		require.NoError(t, err)
		_, err = factory.NewClient(env)
		require.ErrorContains(t, err, "payload validation failed")
	})

	t.Run("valid payload", func(t *testing.T) {
		env, err := hook.GenerateEnvironmentVariables(apphook.Payload{
			BaseURL: "http://localhost", RepoID: 3, PrincipalID: 4, OperationType: enum.GitOpTypeAPIContent,
		})
		require.NoError(t, err)
		client, err := factory.NewClient(env)
		require.NoError(t, err)
		ctrlClient, ok := client.(*ControllerClient)
		require.True(t, ok)
		require.EqualValues(t, 3, ctrlClient.baseInput.RepoID)
		require.EqualValues(t, 4, ctrlClient.baseInput.PrincipalID)
		require.Equal(t, enum.GitOpTypeAPIContent, ctrlClient.baseInput.OperationType)
	})
}

func TestCxrControllerClientCalls(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	base := types.GithookInputBase{
		RepoID: e.repo.ID, PrincipalID: e.admin.ID, OperationType: enum.GitOpTypeGitPush,
	}
	client := &ControllerClient{baseInput: base, githookCtrl: e.ctrl, git: e.git}

	out, err := client.PreReceive(ctx, hook.PreReceiveInput{RefUpdates: []hook.ReferenceUpdate{cxrUpdate(cxrRefFeature)}})
	require.NoError(t, err)
	require.Nil(t, out.Error)

	_, err = client.Update(ctx, hook.UpdateInput{RefUpdate: cxrUpdate(cxrRefFeature)})
	require.NoError(t, err)

	_, err = client.PostReceive(ctx, hook.PostReceiveInput{RefUpdates: []hook.ReferenceUpdate{cxrUpdate(cxrRefMain)}})
	require.NoError(t, err)

	// A missing repository is translated to the hook client not found error.
	missing := &ControllerClient{
		baseInput:   types.GithookInputBase{RepoID: 9999, PrincipalID: e.admin.ID, OperationType: enum.GitOpTypeGitPush},
		githookCtrl: e.ctrl, git: e.git,
	}
	_, err = missing.PreReceive(ctx, hook.PreReceiveInput{})
	require.ErrorIs(t, err, hook.ErrNotFound)
	_, err = missing.Update(ctx, hook.UpdateInput{})
	require.ErrorIs(t, err, hook.ErrNotFound)
	_, err = missing.PostReceive(ctx, hook.PostReceiveInput{})
	require.ErrorIs(t, err, hook.ErrNotFound)

	require.ErrorIs(t, translateControllerError(store.ErrResourceNotFound), hook.ErrNotFound)
	require.ErrorIs(t, translateControllerError(errCxrBoom), errCxrBoom)
}
