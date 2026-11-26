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

package pipeline

import (
	"context"
	"strings"
	"testing"

	pipelineevents "github.com/harness/gitness/app/events/pipeline"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	cxrRepoRef = "root/repo"
	cxrConfig  = ".harness/pipeline.yml"
)

type cxrEnv struct {
	*cxrBase
	repo     *types.Repository
	reporter *pipelineevents.Reporter
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	repo := b.createRepo(ctx, t, b.createSpace(ctx, t, "root"), "repo")

	sys, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "cxr", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	reporter, err := pipelineevents.NewReporter(sys)
	require.NoError(t, err)
	return &cxrEnv{cxrBase: b, repo: repo, reporter: reporter}
}

func (e *cxrEnv) controller(mode cxrAuthMode) *Controller {
	return NewController(
		cxrAuthorizer{mode: mode}, database.NewTriggerStore(e.db), database.NewPipelineStore(e.db),
		*e.reporter, e.repoFinder)
}

func TestCxrPipelineLifecycle(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	created, err := c.Create(ctx, e.session(), cxrRepoRef, &CreateInput{
		UID: "pipe", Description: " d ", ConfigPath: cxrConfig,
	})
	require.NoError(t, err)
	require.Equal(t, "pipe", created.Identifier)
	require.Equal(t, "d", created.Description)

	// The default trigger is created together with the pipeline.
	triggers, err := database.NewTriggerStore(e.db).List(ctx, created.ID, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Len(t, triggers, 1)
	require.Equal(t, "default", triggers[0].Identifier)

	found, err := c.Find(ctx, e.session(), cxrRepoRef, "pipe")
	require.NoError(t, err)
	require.Equal(t, cxrConfig, found.ConfigPath)

	newID, desc, cfg, disabled := "pipe2", " changed ", "other.yml", true
	updated, err := c.Update(ctx, e.session(), cxrRepoRef, "pipe", &UpdateInput{
		Identifier: &newID, Description: &desc, ConfigPath: &cfg, Disabled: &disabled,
	})
	require.NoError(t, err)
	require.Equal(t, newID, updated.Identifier)
	require.Equal(t, "changed", updated.Description)
	require.Equal(t, cfg, updated.ConfigPath)
	require.True(t, updated.Disabled)

	uid := "pipe3"
	updated, err = c.Update(ctx, e.session(), cxrRepoRef, newID, &UpdateInput{UID: &uid})
	require.NoError(t, err)
	require.Equal(t, uid, updated.Identifier)

	require.NoError(t, c.Delete(ctx, e.session(), cxrRepoRef, uid))
	_, err = c.Find(ctx, e.session(), cxrRepoRef, uid)
	require.Error(t, err)
}

func TestCxrPipelineCreateErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)

	tests := []struct {
		name    string
		mode    cxrAuthMode
		repoRef string
		in      *CreateInput
		want    string
	}{
		{"bad identifier", cxrAllow, cxrRepoRef, &CreateInput{Identifier: "!!", ConfigPath: cxrConfig}, ""},
		{"bad description", cxrAllow, cxrRepoRef,
			&CreateInput{Identifier: "a", ConfigPath: cxrConfig, Description: strings.Repeat("x", 1100)}, ""},
		{"missing config path", cxrAllow, cxrRepoRef, &CreateInput{Identifier: "a"}, "config path"},
		{"unknown repo", cxrAllow, "root/nope", &CreateInput{Identifier: "a", ConfigPath: cxrConfig},
			"failed to find repo"},
		{"forbidden", cxrDeny, cxrRepoRef, &CreateInput{Identifier: "a", ConfigPath: cxrConfig},
			"failed to authorize"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.controller(tt.mode).Create(ctx, e.session(), tt.repoRef, tt.in)
			require.Error(t, err)
			require.Nil(t, got)
			require.Contains(t, err.Error(), tt.want)
		})
	}

	c := e.controller(cxrAllow)
	_, err := c.Create(ctx, e.session(), cxrRepoRef, &CreateInput{Identifier: "dup", ConfigPath: cxrConfig})
	require.NoError(t, err)
	_, err = c.Create(ctx, e.session(), cxrRepoRef, &CreateInput{Identifier: "dup", ConfigPath: cxrConfig})
	require.ErrorContains(t, err, "pipeline creation failed")
}

func TestCxrPipelineRepoStateBlocksWrites(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	_, err := e.db.ExecContext(ctx, "UPDATE repositories SET repo_state = $1 WHERE repo_id = $2",
		int(enum.RepoStateGitImport), e.repo.ID)
	require.NoError(t, err)

	_, err = e.controller(cxrAllow).Create(ctx, e.session(), cxrRepoRef, &CreateInput{
		Identifier: "a", ConfigPath: cxrConfig,
	})
	require.Error(t, err)
}

func TestCxrPipelineFindUpdateDeleteErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)
	_, err := allow.Create(ctx, e.session(), cxrRepoRef, &CreateInput{Identifier: "p1", ConfigPath: cxrConfig})
	require.NoError(t, err)

	_, err = allow.Find(ctx, e.session(), "root/nope", "p1")
	require.ErrorContains(t, err, "failed to find repo")
	_, err = deny.Find(ctx, e.session(), cxrRepoRef, "p1")
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Find(ctx, e.session(), cxrRepoRef, "missing")
	require.Error(t, err)

	badID, long, empty := "!!", strings.Repeat("x", 1100), ""
	for _, in := range []*UpdateInput{{Identifier: &badID}, {Description: &long}, {ConfigPath: &empty}} {
		_, err = allow.Update(ctx, e.session(), cxrRepoRef, "p1", in)
		require.ErrorContains(t, err, "failed to sanitize input")
	}
	_, err = deny.Update(ctx, e.session(), cxrRepoRef, "p1", &UpdateInput{})
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Update(ctx, e.session(), cxrRepoRef, "missing", &UpdateInput{})
	require.ErrorContains(t, err, "failed to find pipeline")

	require.ErrorContains(t, deny.Delete(ctx, e.session(), cxrRepoRef, "p1"), "failed to authorize")
	require.ErrorContains(t, allow.Delete(ctx, e.session(), "root/nope", "p1"), "failed to find repo")
}
