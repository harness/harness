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

package scm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type infSecretStore struct {
	store.SecretStore

	secrets map[string]*types.Secret
	err     error
	gotSpc  int64
}

func (s *infSecretStore) FindByIdentifier(_ context.Context, spaceID int64, id string) (*types.Secret, error) {
	s.gotSpc = spaceID
	if s.err != nil {
		return nil, s.err
	}
	sec, ok := s.secrets[id]
	if !ok {
		return nil, errors.New("secret not found")
	}
	return sec, nil
}

func infGithubConnector(apiURL string, auth *types.ConnectorAuth) *types.Connector {
	return &types.Connector{
		SpaceID: 11,
		Type:    enum.ConnectorTypeGithub,
		ConnectorConfig: types.ConnectorConfig{
			Github: &types.GithubConnectorData{APIURL: apiURL, Auth: auth},
		},
	}
}

func infBearerAuth(secret string) *types.ConnectorAuth {
	return &types.ConnectorAuth{
		AuthType: enum.ConnectorAuthTypeBearer,
		Bearer:   &types.BearerTokenCreds{Token: types.SecretRef{Identifier: secret}},
	}
}

func infSecrets() *infSecretStore {
	secrets := map[string]*types.Secret{"gh-token": {Identifier: "gh-token", Data: "token-value-0000"}}
	return &infSecretStore{secrets: secrets}
}

func infGithubServer(t *testing.T, status int) (*httptest.Server, *http.Header) {
	t.Helper()
	var seen http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		if r.URL.Path != "/user" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = w.Write([]byte(`{"id": 1, "login": "alice"}`))
			return
		}
		_, _ = w.Write([]byte(`{"message": "Bad credentials"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

func TestInfService_Test(t *testing.T) {
	ctx := context.Background()

	t.Run("successful connection uses the resolved bearer token", func(t *testing.T) {
		srv, seen := infGithubServer(t, http.StatusOK)
		secrets := infSecrets()
		svc := NewService(secrets, false)

		resp, err := svc.Test(ctx, infGithubConnector(srv.URL+"/", infBearerAuth("gh-token")))
		require.NoError(t, err)
		require.Equal(t, types.ConnectorTestResponse{Status: enum.ConnectorStatusSuccess}, resp)
		require.Equal(t, "Bearer token-value-0000", seen.Get("Authorization"))
		require.EqualValues(t, 11, secrets.gotSpc, "secrets are resolved in the connector's space")
	})
	t.Run("tracing wrapped transport still works", func(t *testing.T) {
		srv, _ := infGithubServer(t, http.StatusOK)
		resp, err := NewService(infSecrets(), true).Test(ctx, infGithubConnector(srv.URL+"/", infBearerAuth("gh-token")))
		require.NoError(t, err)
		require.Equal(t, enum.ConnectorStatusSuccess, resp.Status)
	})
	t.Run("rejected credentials are reported in the response", func(t *testing.T) {
		srv, _ := infGithubServer(t, http.StatusUnauthorized)
		resp, err := NewService(infSecrets(), false).Test(ctx, infGithubConnector(srv.URL+"/", infBearerAuth("gh-token")))
		require.NoError(t, err)
		require.Equal(t, enum.ConnectorStatusFailed, resp.Status)
		require.NotEmpty(t, resp.ErrorMsg)
	})
	t.Run("non scm connectors are rejected", func(t *testing.T) {
		_, err := NewService(infSecrets(), false).Test(ctx, &types.Connector{Type: "docker"})
		require.ErrorContains(t, err, "is not an SCM connector")
	})
	t.Run("provider errors are returned", func(t *testing.T) {
		_, err := NewService(infSecrets(), false).Test(ctx, infGithubConnector("", nil))
		require.ErrorContains(t, err, "github auth needs to be provided")
	})
}

func TestInfGetSCMProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("default github api url", func(t *testing.T) {
		client, err := getSCMProvider(ctx, infGithubConnector("", infBearerAuth("gh-token")), infSecrets(), false)
		require.NoError(t, err)
		require.Equal(t, "https://api.github.com/", client.BaseURL.String())
		require.NotNil(t, client.Client.Transport)
	})
	t.Run("custom api url", func(t *testing.T) {
		client, err := getSCMProvider(ctx, infGithubConnector("https://ghe.example.invalid/api/v3/",
			infBearerAuth("gh-token")), infSecrets(), false)
		require.NoError(t, err)
		require.Equal(t, "https://ghe.example.invalid/api/v3/", client.BaseURL.String())
	})
	t.Run("invalid api url", func(t *testing.T) {
		_, err := getSCMProvider(ctx, infGithubConnector("://bad", infBearerAuth("gh-token")), infSecrets(), false)
		require.Error(t, err)
	})
	t.Run("missing github config", func(t *testing.T) {
		_, err := getSCMProvider(ctx, &types.Connector{Type: enum.ConnectorTypeGithub}, infSecrets(), false)
		require.ErrorContains(t, err, "github connector is nil")
	})
	t.Run("missing auth", func(t *testing.T) {
		_, err := getSCMProvider(ctx, infGithubConnector("", nil), infSecrets(), false)
		require.ErrorContains(t, err, "github auth needs to be provided")
	})
	t.Run("unsupported auth type", func(t *testing.T) {
		auth := &types.ConnectorAuth{AuthType: enum.ConnectorAuthTypeBasic}
		_, err := getSCMProvider(ctx, infGithubConnector("", auth), infSecrets(), false)
		require.ErrorContains(t, err, "unsupported auth type for github connector")
	})
	t.Run("secret lookup failure", func(t *testing.T) {
		_, err := getSCMProvider(ctx, infGithubConnector("", infBearerAuth("missing")), infSecrets(), false)
		require.ErrorContains(t, err, "could not find secret from store")
	})
	t.Run("unsupported provider", func(t *testing.T) {
		_, err := getSCMProvider(ctx, &types.Connector{Type: "gitlab"}, infSecrets(), false)
		require.ErrorContains(t, err, "unsupported scm provider type")
	})
	t.Run("empty token keeps the default transport", func(t *testing.T) {
		secrets := &infSecretStore{secrets: map[string]*types.Secret{"empty": {Identifier: "empty", Data: ""}}}
		client, err := getSCMProvider(ctx, infGithubConnector("", infBearerAuth("empty")), secrets, true)
		require.NoError(t, err)
		require.NotNil(t, client)
	})
}

func TestInfOAuthTransport(t *testing.T) {
	require.Nil(t, oauthTransport("", "Bearer"))
	require.NotNil(t, oauthTransport("token-value-0000", "Bearer"))
	require.NotNil(t, defaultTransport())
}

func TestInfResolveSecret(t *testing.T) {
	secrets := infSecrets()
	got, err := resolveSecret(context.Background(), 7, types.SecretRef{Identifier: "gh-token"}, secrets)
	require.NoError(t, err)
	require.Equal(t, "token-value-0000", got)
	require.EqualValues(t, 7, secrets.gotSpc)

	_, err = resolveSecret(context.Background(), 7, types.SecretRef{Identifier: "nope"}, secrets)
	require.ErrorContains(t, err, "could not find secret from store")
}
