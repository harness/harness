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

package trigger

import (
	"context"
	"strings"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	cxrRepoRef  = "root/repo"
	cxrPipeline = "pipe"
)

type cxrEnv struct {
	*cxrBase
	pipelines store.PipelineStore
	repo      *types.Repository
	pipeline  *types.Pipeline
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	space := b.createSpace(ctx, t, "root")
	repo := b.createRepo(ctx, t, space, "repo")
	e := &cxrEnv{cxrBase: b, repo: repo, pipelines: database.NewPipelineStore(b.db)}
	e.pipeline = &types.Pipeline{
		Identifier: cxrPipeline, RepoID: repo.ID, CreatedBy: b.admin.ID, DefaultBranch: "main", ConfigPath: ".x.yml",
	}
	require.NoError(t, e.pipelines.Create(ctx, e.pipeline))
	return e
}

func (e *cxrEnv) controller(mode cxrAuthMode) *Controller {
	return NewController(cxrAuthorizer{mode: mode}, database.NewTriggerStore(e.db), e.pipelines, e.repoFinder)
}

func TestCxrTriggerCheckHelpers(t *testing.T) {
	require.NoError(t, checkSecret(strings.Repeat("s", triggerMaxSecretLength)))
	require.Error(t, checkSecret(strings.Repeat("s", triggerMaxSecretLength+1)))

	require.NoError(t, checkActions([]enum.TriggerAction{enum.TriggerActionPullReqCreated}))
	require.ErrorContains(t, checkActions([]enum.TriggerAction{"bogus"}), "invalid")

	require.Empty(t, deduplicateActions(nil))
	got := deduplicateActions([]enum.TriggerAction{
		enum.TriggerActionPullReqCreated, enum.TriggerActionPullReqReopened, enum.TriggerActionPullReqCreated,
	})
	require.Equal(t, []enum.TriggerAction{enum.TriggerActionPullReqCreated, enum.TriggerActionPullReqReopened}, got)
}

func TestCxrTriggerLifecycle(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)
	created, err := c.Create(ctx, e.session(), cxrRepoRef, cxrPipeline, &CreateInput{
		UID: "trig", Description: "d", Secret: "s",
		Actions: []enum.TriggerAction{enum.TriggerActionPullReqCreated, enum.TriggerActionPullReqCreated},
	})
	require.NoError(t, err)
	require.Equal(t, "trig", created.Identifier)
	require.Len(t, created.Actions, 1)

	found, err := c.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, "trig")
	require.NoError(t, err)
	require.Equal(t, created.Identifier, found.Identifier)

	list, count, err := c.List(ctx, e.session(), cxrRepoRef, cxrPipeline, types.ListQueryFilter{})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, list, 1)

	desc, hookKey, disabled, newID := " new desc ", "new", true, "trig2"
	updated, err := c.Update(ctx, e.session(), cxrRepoRef, cxrPipeline, "trig", &UpdateInput{
		Identifier: &newID, Description: &desc, Secret: &hookKey, Disabled: &disabled,
		Actions: []enum.TriggerAction{enum.TriggerActionPullReqMerged},
	})
	require.NoError(t, err)
	require.Equal(t, newID, updated.Identifier)
	require.Equal(t, "new desc", updated.Description)
	require.Equal(t, hookKey, updated.Secret)
	require.True(t, updated.Disabled)
	require.Equal(t, []enum.TriggerAction{enum.TriggerActionPullReqMerged}, updated.Actions)

	uid := "trig3"
	updated, err = c.Update(ctx, e.session(), cxrRepoRef, cxrPipeline, newID, &UpdateInput{UID: &uid})
	require.NoError(t, err)
	require.Equal(t, uid, updated.Identifier)

	require.NoError(t, c.Delete(ctx, e.session(), cxrRepoRef, cxrPipeline, uid))
	_, err = c.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, uid)
	require.ErrorContains(t, err, "failed to find trigger")
}

func TestCxrTriggerCreateErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	long := strings.Repeat("x", 1100)

	tests := []struct {
		name     string
		mode     cxrAuthMode
		repoRef  string
		pipeline string
		in       *CreateInput
		want     string
	}{
		{"bad description", cxrAllow, cxrRepoRef, cxrPipeline, &CreateInput{Identifier: "a", Description: long}, ""},
		{"bad secret", cxrAllow, cxrRepoRef, cxrPipeline,
			&CreateInput{Identifier: "a", Secret: strings.Repeat("s", triggerMaxSecretLength+1)}, "secret"},
		{"bad action", cxrAllow, cxrRepoRef, cxrPipeline,
			&CreateInput{Identifier: "a", Actions: []enum.TriggerAction{"bogus"}}, "invalid"},
		{"bad identifier", cxrAllow, cxrRepoRef, cxrPipeline, &CreateInput{Identifier: "!!"}, "invalid input"},
		{"unknown repo", cxrAllow, "root/nope", cxrPipeline, &CreateInput{Identifier: "a"}, "failed to find repo"},
		{"forbidden", cxrDeny, cxrRepoRef, cxrPipeline, &CreateInput{Identifier: "a"}, "failed to authorize"},
		{"unknown pipeline", cxrAllow, cxrRepoRef, "nope", &CreateInput{Identifier: "a"}, "failed to find pipeline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.controller(tt.mode).Create(ctx, e.session(), tt.repoRef, tt.pipeline, tt.in)
			require.Error(t, err)
			require.Nil(t, got)
			require.Contains(t, err.Error(), tt.want)
		})
	}

	c := e.controller(cxrAllow)
	_, err := c.Create(ctx, e.session(), cxrRepoRef, cxrPipeline, &CreateInput{Identifier: "dup"})
	require.NoError(t, err)
	_, err = c.Create(ctx, e.session(), cxrRepoRef, cxrPipeline, &CreateInput{Identifier: "dup"})
	require.ErrorContains(t, err, "trigger creation failed")
}

func TestCxrTriggerRepoStateBlocksWrites(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	_, err := e.db.ExecContext(ctx, "UPDATE repositories SET repo_state = $1 WHERE repo_id = $2",
		int(enum.RepoStateGitImport), e.repo.ID)
	require.NoError(t, err)
	c := e.controller(cxrAllow)

	_, err = c.Create(ctx, e.session(), cxrRepoRef, cxrPipeline, &CreateInput{Identifier: "a"})
	require.Error(t, err)
	require.Error(t, c.Delete(ctx, e.session(), cxrRepoRef, cxrPipeline, "a"))
}

func TestCxrTriggerReadWriteErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)
	_, err := allow.Create(ctx, e.session(), cxrRepoRef, cxrPipeline, &CreateInput{Identifier: "t1"})
	require.NoError(t, err)

	_, err = allow.Find(ctx, e.session(), "root/nope", cxrPipeline, "t1")
	require.ErrorContains(t, err, "failed to find repo")
	_, err = deny.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, "t1")
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Find(ctx, e.session(), cxrRepoRef, "nope", "t1")
	require.ErrorContains(t, err, "failed to find pipeline")
	_, err = allow.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, "missing")
	require.ErrorContains(t, err, "failed to find trigger")

	_, _, err = deny.List(ctx, e.session(), cxrRepoRef, cxrPipeline, types.ListQueryFilter{})
	require.ErrorContains(t, err, "failed to authorize")
	_, _, err = allow.List(ctx, e.session(), cxrRepoRef, "nope", types.ListQueryFilter{})
	require.ErrorContains(t, err, "failed to find pipeline")

	badID, long, badSecret := "!!", strings.Repeat("x", 1100), strings.Repeat("s", triggerMaxSecretLength+1)
	for _, in := range []*UpdateInput{
		{Identifier: &badID},
		{Description: &long},
		{Secret: &badSecret},
		{Actions: []enum.TriggerAction{"bogus"}},
	} {
		_, err = allow.Update(ctx, e.session(), cxrRepoRef, cxrPipeline, "t1", in)
		require.ErrorContains(t, err, "invalid input")
	}
	_, err = deny.Update(ctx, e.session(), cxrRepoRef, cxrPipeline, "t1", &UpdateInput{})
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Update(ctx, e.session(), cxrRepoRef, "nope", "t1", &UpdateInput{})
	require.ErrorContains(t, err, "failed to find pipeline")
	_, err = allow.Update(ctx, e.session(), cxrRepoRef, cxrPipeline, "missing", &UpdateInput{})
	require.ErrorContains(t, err, "failed to find trigger")

	require.ErrorContains(t, deny.Delete(ctx, e.session(), cxrRepoRef, cxrPipeline, "t1"), "failed to authorize")
	require.ErrorContains(t, allow.Delete(ctx, e.session(), cxrRepoRef, "nope", "t1"), "failed to find pipeline")
}
