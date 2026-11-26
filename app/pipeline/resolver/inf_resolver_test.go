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

package resolver

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	v1yaml "github.com/drone/spec/dist/go"
	"github.com/stretchr/testify/require"
)

const infPluginSpec = `kind: plugin
type: step
name: slack
spec:
  description: Send a message to slack
  step:
    type: script
    spec:
      run: echo hello
`

const infStagePluginSpec = `kind: plugin
type: stage
name: stage-plugin
spec:
  description: A stage plugin
  stage:
    type: ci
    spec:
      steps: []
`

const infTemplateSpec = `kind: template
type: step
name: my-template
spec:
  step:
    type: script
    spec:
      run: echo from template
`

type infPluginStore struct {
	store.PluginStore

	plugins   map[string]*types.Plugin
	listErr   error
	findErr   error
	createErr error
	updateErr error
	created   []*types.Plugin
	updated   []*types.Plugin
}

func (s *infPluginStore) Find(_ context.Context, name, _ string) (*types.Plugin, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	p, ok := s.plugins[name]
	if !ok {
		return nil, errors.New("plugin not found")
	}
	return p, nil
}

func (s *infPluginStore) ListAll(context.Context) ([]*types.Plugin, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	res := make([]*types.Plugin, 0, len(s.plugins))
	for _, p := range s.plugins {
		res = append(res, p)
	}
	return res, nil
}

func (s *infPluginStore) Create(_ context.Context, p *types.Plugin) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, p)
	return nil
}

func (s *infPluginStore) Update(_ context.Context, p *types.Plugin) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, p)
	return nil
}

type infTemplateStore struct {
	store.TemplateStore

	templates map[string]*types.Template
	gotSpace  int64
	gotType   enum.ResolverType
}

func (s *infTemplateStore) FindByIdentifierAndType(
	_ context.Context,
	spaceID int64,
	identifier string,
	t enum.ResolverType,
) (*types.Template, error) {
	s.gotSpace = spaceID
	s.gotType = t
	tpl, ok := s.templates[identifier]
	if !ok {
		return nil, errors.New("template not found")
	}
	return tpl, nil
}

type infExecutionStore struct {
	store.ExecutionStore

	execution *types.Execution
	err       error
}

func (s *infExecutionStore) Find(context.Context, int64) (*types.Execution, error) {
	return s.execution, s.err
}

type infRepoStore struct {
	store.RepoStore

	repo *types.Repository
	err  error
}

func (s *infRepoStore) Find(context.Context, int64) (*types.Repository, error) {
	return s.repo, s.err
}

func TestInfResolve(t *testing.T) {
	ctx := context.Background()
	plugins := &infPluginStore{plugins: map[string]*types.Plugin{
		"slack":      {Identifier: "slack", Spec: infPluginSpec},
		"broken":     {Identifier: "broken", Spec: "this: is: not: valid"},
		"stage-only": {Identifier: "stage-only", Spec: infStagePluginSpec},
	}}
	templates := &infTemplateStore{templates: map[string]*types.Template{
		"my-template": {Identifier: "my-template", Data: infTemplateSpec},
		"bad":         {Identifier: "bad", Data: "{{{"},
	}}
	resolve := Resolve(ctx, plugins, templates, 42)

	t.Run("plugin", func(t *testing.T) {
		cfg, err := resolve("slack", "plugin", "step", "")
		require.NoError(t, err)
		require.Equal(t, "slack", cfg.Name)
		_, ok := cfg.Spec.(*v1yaml.PluginStep)
		require.True(t, ok)
	})
	t.Run("template", func(t *testing.T) {
		cfg, err := resolve("my-template", "template", "step", "")
		require.NoError(t, err)
		require.Equal(t, "my-template", cfg.Name)
		require.EqualValues(t, 42, templates.gotSpace)
		require.Equal(t, enum.ResolverTypeStep, templates.gotType)
	})
	t.Run("invalid kind", func(t *testing.T) {
		_, err := resolve("x", "bogus", "step", "")
		require.Error(t, err)
	})
	t.Run("invalid type", func(t *testing.T) {
		_, err := resolve("x", "plugin", "bogus", "")
		require.Error(t, err)
	})
	t.Run("stage level plugins are unsupported", func(t *testing.T) {
		_, err := resolve("stage-only", "plugin", "stage", "")
		require.ErrorContains(t, err, "only step level plugins are currently supported")
	})
	t.Run("plugin lookup failure", func(t *testing.T) {
		_, err := resolve("missing", "plugin", "step", "")
		require.ErrorContains(t, err, "could not lookup plugin")
	})
	t.Run("plugin with invalid spec", func(t *testing.T) {
		_, err := resolve("broken", "plugin", "step", "")
		require.ErrorContains(t, err, "could not unmarshal plugin")
	})
	t.Run("template lookup failure", func(t *testing.T) {
		_, err := resolve("missing", "template", "step", "")
		require.ErrorContains(t, err, "could not find template")
	})
	t.Run("template with invalid spec", func(t *testing.T) {
		_, err := resolve("bad", "template", "step", "")
		require.ErrorContains(t, err, "could not unmarshal template")
	})
}

func TestInfManager_GetLookupFn(t *testing.T) {
	templates := &infTemplateStore{templates: map[string]*types.Template{
		"my-template": {Identifier: "my-template", Data: infTemplateSpec},
	}}

	t.Run("resolves in the repo's space", func(t *testing.T) {
		m := NewManager(&types.Config{}, &infPluginStore{}, templates,
			&infExecutionStore{execution: &types.Execution{RepoID: 3}},
			&infRepoStore{repo: &types.Repository{ParentID: 77}})
		cfg, err := m.GetLookupFn()("my-template", "template", "step", "", 1)
		require.NoError(t, err)
		require.Equal(t, "my-template", cfg.Name)
		require.EqualValues(t, 77, templates.gotSpace)
	})
	t.Run("execution not found", func(t *testing.T) {
		m := NewManager(&types.Config{}, nil, nil, &infExecutionStore{err: errors.New("nope")}, nil)
		_, err := m.GetLookupFn()("x", "template", "step", "", 1)
		require.ErrorContains(t, err, "could not find relevant execution")
	})
	t.Run("repo not found", func(t *testing.T) {
		m := NewManager(&types.Config{}, nil, nil,
			&infExecutionStore{execution: &types.Execution{}}, &infRepoStore{err: errors.New("nope")})
		_, err := m.GetLookupFn()("x", "template", "step", "", 1)
		require.ErrorContains(t, err, "could not find relevant repo")
	})
}

type infZipEntry struct {
	name string
	data string
}

func infWriteZip(t *testing.T, entries ...infZipEntry) string {
	t.Helper()
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		require.NoError(t, err)
		_, err = w.Write([]byte(e.data))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	path := filepath.Join(t.TempDir(), "plugins.zip")
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o600))
	return path
}

func TestInfManager_Populate(t *testing.T) {
	ctx := context.Background()
	zipPath := infWriteZip(t,
		infZipEntry{"repo-master/plugins/slack/slack.yaml", infPluginSpec},
		infZipEntry{"repo-master/plugins/slack/logo.svg", "<svg/>"},
		infZipEntry{"repo-master/plugins/broken/broken.yaml", "this: is: not: valid"},
		infZipEntry{"repo-master/plugins/stage/stage.yaml", infStagePluginSpec},
		infZipEntry{"repo-master/readme.md", "ignored"},
		infZipEntry{"repo-master/other/dir/file.yaml", "ignored"},
	)

	t.Run("creates new plugins with logo", func(t *testing.T) {
		ps := &infPluginStore{plugins: map[string]*types.Plugin{}}
		m := NewManager(infConfig(zipPath), ps, nil, nil, nil)
		require.NoError(t, m.Populate(ctx))
		require.Len(t, ps.created, 2)
		byName := map[string]*types.Plugin{}
		for _, p := range ps.created {
			byName[p.Identifier] = p
		}
		slack := byName["slack"]
		require.NotNil(t, slack)
		require.Equal(t, "Send a message to slack", slack.Description)
		require.Equal(t, "<svg/>", slack.Logo)
		require.Equal(t, infPluginSpec, slack.Spec)
		require.Equal(t, "step", slack.Type)
		require.Empty(t, ps.updated)
	})
	t.Run("identical plugins are left alone", func(t *testing.T) {
		ps := &infPluginStore{plugins: map[string]*types.Plugin{
			"slack": {
				Identifier: "slack", Description: "Send a message to slack", Spec: infPluginSpec, Logo: "<svg/>",
			},
			"stage-plugin": {
				Identifier: "stage-plugin", Description: "A stage plugin", Spec: infStagePluginSpec,
			},
		}}
		m := NewManager(infConfig(zipPath), ps, nil, nil, nil)
		require.NoError(t, m.Populate(ctx))
		require.Empty(t, ps.created)
		require.Empty(t, ps.updated)
	})
	t.Run("changed plugins are updated", func(t *testing.T) {
		ps := &infPluginStore{plugins: map[string]*types.Plugin{
			"slack":        {Identifier: "slack", Description: "stale", Spec: "old"},
			"stage-plugin": {Identifier: "stage-plugin", Description: "A stage plugin", Spec: infStagePluginSpec},
		}}
		m := NewManager(infConfig(zipPath), ps, nil, nil, nil)
		require.NoError(t, m.Populate(ctx))
		require.Len(t, ps.updated, 1)
		require.Equal(t, "slack", ps.updated[0].Identifier)
	})
	t.Run("store write failures are tolerated", func(t *testing.T) {
		ps := &infPluginStore{
			plugins:   map[string]*types.Plugin{"slack": {Identifier: "slack", Spec: "old"}},
			createErr: errors.New("create failed"),
			updateErr: errors.New("update failed"),
		}
		m := NewManager(infConfig(zipPath), ps, nil, nil, nil)
		require.NoError(t, m.Populate(ctx))
		require.Empty(t, ps.created)
		require.Empty(t, ps.updated)
	})
	t.Run("list failure", func(t *testing.T) {
		ps := &infPluginStore{listErr: errors.New("list failed")}
		m := NewManager(infConfig(zipPath), ps, nil, nil, nil)
		require.ErrorContains(t, m.Populate(ctx), "could not list plugins")
	})
	t.Run("no url configured", func(t *testing.T) {
		m := NewManager(&types.Config{}, nil, nil, nil, nil)
		require.ErrorContains(t, m.Populate(ctx), "plugins url not provided")
	})
	t.Run("not a zip file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "plugins.zip")
		require.NoError(t, os.WriteFile(path, []byte("not a zip"), 0o600))
		m := NewManager(infConfig(path), &infPluginStore{}, nil, nil, nil)
		require.ErrorContains(t, m.Populate(ctx), "could not open zip for reading")
	})
}

func TestInfManager_PopulateFromRemoteURL(t *testing.T) {
	zipPath := infWriteZip(t, infZipEntry{"repo-master/plugins/slack/slack.yaml", infPluginSpec})
	data, err := os.ReadFile(zipPath)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/plugins.zip" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	ps := &infPluginStore{plugins: map[string]*types.Plugin{}}
	m := NewManager(infConfig(srv.URL+"/plugins.zip"), ps, nil, nil, nil)
	require.NoError(t, m.Populate(context.Background()))
	require.Len(t, ps.created, 1)

	// a body that is not a zip is reported when opening it.
	m = NewManager(infConfig(srv.URL+"/missing"), ps, nil, nil, nil)
	require.ErrorContains(t, m.Populate(context.Background()), "could not open zip for reading")

	// an unreachable server is reported as a download failure.
	url := srv.URL + "/plugins.zip"
	srv.Close()
	m = NewManager(infConfig(url), ps, nil, nil, nil)
	require.ErrorContains(t, m.Populate(context.Background()), "could not download remote zip")
}

func TestInfDownloadZip_InvalidURL(t *testing.T) {
	err := downloadZip(context.Background(), "://bad url", filepath.Join(t.TempDir(), "x"))
	require.ErrorContains(t, err, "failed to create request")
}

func infConfig(zipURL string) *types.Config {
	c := &types.Config{}
	c.CI.PluginsZipURL = zipURL
	return c
}
