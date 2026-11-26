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

package connector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/connector"
	"github.com/harness/gitness/app/connector/scm"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const cxrSecretID = "gh-token"

type cxrEnv struct {
	*cxrBase
	secrets store.SecretStore
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	space := b.createSpace(ctx, t, "root")
	e := &cxrEnv{cxrBase: b, secrets: database.NewSecretStore(b.db)}
	require.NoError(t, e.secrets.Create(ctx, &types.Secret{
		Identifier: cxrSecretID, SpaceID: space.ID, CreatedBy: b.admin.ID, Data: "token-value-0000",
	}))
	return e
}

func (e *cxrEnv) controller(mode cxrAuthMode) *Controller {
	return NewController(
		cxrAuthorizer{mode: mode}, database.NewConnectorStore(e.db, e.secrets),
		connector.New(e.secrets, scm.NewService(e.secrets, false)), e.spaceFinder)
}

func cxrGithubConfig(apiURL string) types.ConnectorConfig {
	return types.ConnectorConfig{Github: &types.GithubConnectorData{
		APIURL: apiURL,
		Auth: &types.ConnectorAuth{
			AuthType: enum.ConnectorAuthTypeBearer,
			Bearer:   &types.BearerTokenCreds{Token: types.SecretRef{Identifier: cxrSecretID}},
		},
	}}
}

func cxrCreateInput(apiURL string) *CreateInput {
	return &CreateInput{
		SpaceRef: "root", Identifier: "gh", Type: enum.ConnectorTypeGithub,
		ConnectorConfig: cxrGithubConfig(apiURL),
	}
}

func TestCxrConnectorLifecycle(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	in := cxrCreateInput("https://api.example.com/")
	in.Description = "  desc  "
	created, err := c.Create(ctx, e.session(), in)
	require.NoError(t, err)
	require.Equal(t, "desc", created.Description)

	found, err := c.Find(ctx, e.session(), "root", "gh")
	require.NoError(t, err)
	require.Equal(t, enum.ConnectorTypeGithub, found.Type)
	require.Equal(t, "https://api.example.com/", found.Github.APIURL)

	newID, newDesc := "gh2", "changed"
	cfg := cxrGithubConfig("https://other.example.com/")
	updated, err := c.Update(ctx, e.session(), "root", "gh", &UpdateInput{
		Identifier: &newID, Description: &newDesc, ConnectorConfig: &cfg,
	})
	require.NoError(t, err)
	require.Equal(t, newID, updated.Identifier)
	require.Equal(t, newDesc, updated.Description)
	require.Equal(t, "https://other.example.com/", updated.Github.APIURL)

	require.NoError(t, c.Delete(ctx, e.session(), "root", newID))
	_, err = c.Find(ctx, e.session(), "root", newID)
	require.ErrorContains(t, err, "failed to find connector")
}

func TestCxrConnectorCreateErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)

	mutate := func(fn func(in *CreateInput)) *CreateInput {
		in := cxrCreateInput("https://api.example.com/")
		fn(in)
		return in
	}
	tests := []struct {
		name string
		mode cxrAuthMode
		in   *CreateInput
		want string
	}{
		{"empty space ref", cxrAllow, mutate(func(in *CreateInput) { in.SpaceRef = " " }), "Parent space required"},
		{"zero space id", cxrAllow, mutate(func(in *CreateInput) { in.SpaceRef = "0" }), "Parent space required"},
		{"invalid type", cxrAllow, mutate(func(in *CreateInput) { in.Type = "bogus" }), "Invalid connector type"},
		{"missing config", cxrAllow, mutate(func(in *CreateInput) { in.Github = nil }), "invalid connector config"},
		{"bad identifier", cxrAllow, mutate(func(in *CreateInput) { in.Identifier = "!!" }), ""},
		{"bad description", cxrAllow, mutate(func(in *CreateInput) { in.Description = strings.Repeat("x", 1100) }), ""},
		{"unknown space", cxrAllow, mutate(func(in *CreateInput) { in.SpaceRef = "nope" }), "failed to find parent"},
		{"forbidden", cxrDeny, mutate(func(*CreateInput) {}), ""},
		{"unknown secret", cxrAllow, mutate(func(in *CreateInput) {
			in.Github.Auth.Bearer.Token.Identifier = "missing"
		}), "connector creation failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.controller(tt.mode).Create(ctx, e.session(), tt.in)
			require.Error(t, err)
			require.Nil(t, got)
			require.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestCxrConnectorFindUpdateDeleteErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)
	_, err := allow.Create(ctx, e.session(), cxrCreateInput("https://api.example.com/"))
	require.NoError(t, err)

	_, err = allow.Find(ctx, e.session(), "nope", "gh")
	require.ErrorContains(t, err, "failed to find space")
	_, err = deny.Find(ctx, e.session(), "root", "gh")
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Find(ctx, e.session(), "root", "missing")
	require.ErrorContains(t, err, "failed to find connector")

	badID, long := "!!", strings.Repeat("x", 1100)
	_, err = allow.Update(ctx, e.session(), "root", "gh", &UpdateInput{Identifier: &badID})
	require.ErrorContains(t, err, "failed to sanitize input")
	_, err = allow.Update(ctx, e.session(), "root", "gh", &UpdateInput{Description: &long})
	require.ErrorContains(t, err, "failed to sanitize input")
	_, err = allow.Update(ctx, e.session(), "nope", "gh", &UpdateInput{})
	require.ErrorContains(t, err, "failed to find space")
	_, err = deny.Update(ctx, e.session(), "root", "gh", &UpdateInput{})
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Update(ctx, e.session(), "root", "missing", &UpdateInput{})
	require.ErrorContains(t, err, "failed to find connector")
	_, err = allow.Update(ctx, e.session(), "root", "gh", &UpdateInput{ConnectorConfig: &types.ConnectorConfig{}})
	require.ErrorContains(t, err, "Failed to validate connector config")

	require.ErrorContains(t, allow.Delete(ctx, e.session(), "nope", "gh"), "failed to find space")
	require.ErrorContains(t, deny.Delete(ctx, e.session(), "root", "gh"), "failed to authorize")
}

func TestCxrConnectorTest(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)

	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"id": 1, "login": "alice"}`))
	}))
	t.Cleanup(srv.Close)

	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)
	_, err := allow.Create(ctx, e.session(), cxrCreateInput(srv.URL+"/"))
	require.NoError(t, err)

	resp, err := allow.Test(ctx, e.session(), "root", "gh")
	require.NoError(t, err)
	require.Equal(t, enum.ConnectorStatusSuccess, resp.Status)

	// The last test result is persisted on the connector.
	found, err := allow.Find(ctx, e.session(), "root", "gh")
	require.NoError(t, err)
	require.Equal(t, enum.ConnectorStatusSuccess, found.LastTestStatus)
	require.NotZero(t, found.LastTestAttempt)

	status = http.StatusUnauthorized
	resp, err = allow.Test(ctx, e.session(), "root", "gh")
	require.NoError(t, err)
	require.Equal(t, enum.ConnectorStatusFailed, resp.Status)

	_, err = allow.Test(ctx, e.session(), "nope", "gh")
	require.ErrorContains(t, err, "failed to find space")
	_, err = deny.Test(ctx, e.session(), "root", "gh")
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Test(ctx, e.session(), "root", "missing")
	require.ErrorContains(t, err, "failed to find connector")
}
