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

package infraprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errCxrFactory = errors.New("factory failure")

// cxrProvider is a minimal infraprovider.Provider.
type cxrProvider struct {
	validateErr error
}

func (cxrProvider) UpdateParams(
	params []types.InfraProviderParameter, _ map[string]any,
) ([]types.InfraProviderParameter, error) {
	return params, nil
}

func (cxrProvider) ValidateParams([]types.InfraProviderParameter) error { return nil }

func (cxrProvider) TemplateParams() []types.InfraProviderParameterSchema { return nil }

func (cxrProvider) UpdateConfig(c *types.InfraProviderConfig) (*types.InfraProviderConfig, error) {
	return c, nil
}

func (p cxrProvider) ValidateConfig(*types.InfraProviderConfig) error { return p.validateErr }

func (cxrProvider) GenerateSetupYAML(c *types.InfraProviderConfig) (string, error) {
	return "setup: " + c.Identifier, nil
}

type cxrFactory struct {
	provider cxrProvider
	err      error
}

func (f *cxrFactory) GetInfraProvider(enum.InfraProviderType) (infraprovider.Provider, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.provider, nil
}

// cxrGitspaceConfigs reports a configurable number of active gitspaces.
type cxrGitspaceConfigs struct {
	store.GitspaceConfigStore
	active []*types.GitspaceConfig
}

func (s *cxrGitspaceConfigs) ListActiveConfigsForInfraProviderResource(
	context.Context, int64,
) ([]*types.GitspaceConfig, error) {
	return s.active, nil
}

type cxrEnv struct {
	*cxrBase
	factory   *cxrFactory
	gitspaces *cxrGitspaceConfigs
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	b.createSpace(ctx, t, "root")
	return &cxrEnv{cxrBase: b, factory: &cxrFactory{}, gitspaces: &cxrGitspaceConfigs{}}
}

func (e *cxrEnv) controller(mode cxrAuthMode) *Controller {
	svc := infraprovider.NewService(
		dbtx.New(e.db), e.gitspaces, database.NewInfraProviderResourceStore(e.db, e.spaceIDs),
		database.NewInfraProviderConfigStore(e.db, e.spaceIDs), database.NewInfraProviderTemplateStore(e.db),
		e.factory, e.spaceFinder, nil)
	return NewController(cxrAuthorizer{mode: mode}, e.spaceFinder, svc)
}

func cxrConfigInput() ConfigInput {
	return ConfigInput{
		Identifier: "docker1", SpaceRef: "root", Name: "Docker", Type: enum.InfraProviderTypeDocker,
		Metadata: map[string]any{},
	}
}

func cxrResourceInput(identifier string) ResourceInput {
	disk := "10"
	return ResourceInput{
		Identifier: identifier, Name: identifier, InfraProviderType: enum.InfraProviderTypeDocker,
		Region: "local", Disk: &disk, Metadata: map[string]string{"cpu": "2", "memory": "4"},
	}
}

func TestCxrInfraProviderConfigLifecycle(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	created, err := c.CreateConfig(ctx, *e.session(), cxrConfigInput())
	require.NoError(t, err)
	require.Equal(t, "docker1", created.Identifier)
	require.Equal(t, "root", created.SpacePath)

	found, err := c.Find(ctx, e.session(), "root", "docker1")
	require.NoError(t, err)
	require.Equal(t, "setup: docker1", found.SetupYAML)
	require.Empty(t, found.Resources)

	configs, err := c.List(ctx, e.session(), "root", false)
	require.NoError(t, err)
	require.Len(t, configs, 1)

	require.NoError(t, c.DeleteConfig(ctx, e.session(), "root", "docker1"))
	_, err = c.Find(ctx, e.session(), "root", "docker1")
	require.ErrorContains(t, err, "failed to find infraprovider config")
}

func TestCxrInfraProviderConfigErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)

	bad := cxrConfigInput()
	bad.Identifier = "!!"
	_, err := allow.CreateConfig(ctx, *e.session(), bad)
	require.ErrorContains(t, err, "invalid input")

	unknown := cxrConfigInput()
	unknown.SpaceRef = "nope"
	_, err = allow.CreateConfig(ctx, *e.session(), unknown)
	require.ErrorContains(t, err, "failed to find parent by ref")

	_, err = deny.CreateConfig(ctx, *e.session(), cxrConfigInput())
	require.Error(t, err)

	e.factory.provider.validateErr = errCxrFactory
	_, err = allow.CreateConfig(ctx, *e.session(), cxrConfigInput())
	require.ErrorIs(t, err, errCxrFactory)
	e.factory.provider.validateErr = nil

	// Only one config per provider type and space is allowed.
	_, err = allow.CreateConfig(ctx, *e.session(), cxrConfigInput())
	require.NoError(t, err)
	second := cxrConfigInput()
	second.Identifier = "docker2"
	_, err = allow.CreateConfig(ctx, *e.session(), second)
	require.ErrorContains(t, err, "Only 1 is allowed")

	_, err = allow.Find(ctx, e.session(), "nope", "docker1")
	require.ErrorContains(t, err, "failed to find space")

	_, err = allow.List(ctx, e.session(), "nope", false)
	require.ErrorContains(t, err, "failed to find space")
	_, err = deny.List(ctx, e.session(), "root", false)
	require.ErrorContains(t, err, "failed to authorize")

	require.ErrorContains(t, allow.DeleteConfig(ctx, e.session(), "nope", "docker1"), "failed to find space")
	require.ErrorContains(t, deny.DeleteConfig(ctx, e.session(), "root", "docker1"), "failed to authorize")
	require.ErrorContains(t, allow.DeleteConfig(ctx, e.session(), "root", "missing"), "could not find infra provider")

	// Provider lookup failures surface while populating details.
	e.factory.err = errCxrFactory
	_, err = allow.Find(ctx, e.session(), "root", "docker1")
	require.ErrorIs(t, err, errCxrFactory)
}

func TestCxrInfraProviderResources(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)
	_, err := c.CreateConfig(ctx, *e.session(), cxrConfigInput())
	require.NoError(t, err)

	resources, err := c.CreateResources(ctx, *e.session(), []ResourceInput{
		cxrResourceInput("res1"), cxrResourceInput("res2"),
	}, "docker1", "root")
	require.NoError(t, err)
	require.Len(t, resources, 2)

	listed, err := c.ListResources(ctx, e.session(), "root", "docker1")
	require.NoError(t, err)
	require.Len(t, listed, 2)

	found, err := c.Find(ctx, e.session(), "root", "docker1")
	require.NoError(t, err)
	require.Len(t, found.Resources, 2)

	// Resources of a config block its deletion.
	require.ErrorContains(t, c.DeleteConfig(ctx, e.session(), "root", "docker1"), "resources in this config")

	// Duplicates are rejected.
	_, err = c.CreateResources(ctx, *e.session(), []ResourceInput{cxrResourceInput("res1")}, "docker1", "root")
	require.ErrorContains(t, err, "already exists")

	root, err := e.spaceFinder.FindByRef(ctx, "root")
	require.NoError(t, err)

	// Active gitspaces block the deletion of their resource.
	e.gitspaces.active = []*types.GitspaceConfig{{}}
	err = c.DeleteResource(ctx, e.session(), root.ID, "docker1", "res1")
	require.ErrorContains(t, err, "active gitspace")
	e.gitspaces.active = nil

	require.NoError(t, c.DeleteResource(ctx, e.session(), root.ID, "docker1", "res1"))
	require.NoError(t, c.DeleteResource(ctx, e.session(), root.ID, "docker1", "res2"))
	require.ErrorContains(t,
		c.DeleteResource(ctx, e.session(), root.ID, "docker1", "res2"), "failed to find infra resource")
	require.ErrorContains(t, c.DeleteResource(ctx, e.session(), root.ID, "missing", "res2"), "failed to find infra config")
	require.NoError(t, c.DeleteConfig(ctx, e.session(), "root", "docker1"))
}

func TestCxrInfraProviderResourceErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)
	_, err := allow.CreateConfig(ctx, *e.session(), cxrConfigInput())
	require.NoError(t, err)
	root, err := e.spaceFinder.FindByRef(ctx, "root")
	require.NoError(t, err)

	_, err = allow.CreateResources(ctx, *e.session(), []ResourceInput{cxrResourceInput("!!")}, "docker1", "root")
	require.ErrorContains(t, err, "invalid input")
	_, err = allow.CreateResources(ctx, *e.session(), []ResourceInput{cxrResourceInput("r")}, "docker1", "nope")
	require.ErrorContains(t, err, "failed to find parent by ref")
	_, err = deny.CreateResources(ctx, *e.session(), []ResourceInput{cxrResourceInput("r")}, "docker1", "root")
	require.Error(t, err)
	_, err = allow.CreateResources(ctx, *e.session(), []ResourceInput{cxrResourceInput("r")}, "missing", "root")
	require.ErrorContains(t, err, "failed to find config")

	_, err = allow.ListResources(ctx, e.session(), "nope", "docker1")
	require.ErrorContains(t, err, "failed to find space")
	_, err = deny.ListResources(ctx, e.session(), "root", "docker1")
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.ListResources(ctx, e.session(), "root", "missing")
	require.ErrorContains(t, err, "failed to find infra provider")

	err = allow.DeleteResource(ctx, e.session(), 99999, "docker1", "r")
	require.ErrorContains(t, err, "failed to find space")
	err = deny.DeleteResource(ctx, e.session(), root.ID, "docker1", "r")
	require.ErrorContains(t, err, "failed to authorize")
}

func TestCxrInfraProviderTemplates(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)
	_, err := allow.CreateConfig(ctx, *e.session(), cxrConfigInput())
	require.NoError(t, err)

	tpl, err := allow.CreateTemplate(ctx, e.session(), &TemplateInput{
		Identifier: "tpl", Description: "d", Data: "data",
	}, "docker1", "root")
	require.NoError(t, err)
	require.Equal(t, "tpl", tpl.Identifier)
	require.Equal(t, "docker1", tpl.InfraProviderConfigIdentifier)

	in := &TemplateInput{Identifier: "tpl2"}
	_, err = allow.CreateTemplate(ctx, e.session(), in, "docker1", "nope")
	require.ErrorContains(t, err, "failed to find parent by ref")
	_, err = deny.CreateTemplate(ctx, e.session(), in, "docker1", "root")
	require.Error(t, err)
	_, err = allow.CreateTemplate(ctx, e.session(), in, "missing", "root")
	require.ErrorContains(t, err, "failed to find infraprovider config by ref")
	_, err = allow.CreateTemplate(ctx, e.session(), &TemplateInput{Identifier: "tpl"}, "docker1", "root")
	require.Error(t, err, "duplicate template identifier")
}
