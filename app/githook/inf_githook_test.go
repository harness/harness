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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestInfPayload_Validate(t *testing.T) {
	valid := Payload{BaseURL: "http://host/hooks", RepoID: 1, PrincipalID: 2}
	tests := []struct {
		name    string
		mutate  func(p *Payload)
		wantErr string
	}{
		{name: "valid", mutate: func(*Payload) {}},
		{name: "no base url", mutate: func(p *Payload) { p.BaseURL = "" }, wantErr: "base url"},
		{name: "no principal", mutate: func(p *Payload) { p.PrincipalID = 0 }, wantErr: "principal id"},
		{name: "negative repo", mutate: func(p *Payload) { p.RepoID = -1 }, wantErr: "repo id"},
		{name: "disabled skips validation", mutate: func(p *Payload) { *p = Payload{Disabled: true} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid
			tt.mutate(&p)
			err := p.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestInfGetInputBaseFromPayload(t *testing.T) {
	tests := []struct {
		name         string
		payload      Payload
		wantOp       enum.GitOpType
		wantInternal bool
	}{
		{"explicit git push", Payload{OperationType: enum.GitOpTypeGitPush}, enum.GitOpTypeGitPush, false},
		{"explicit api content", Payload{OperationType: enum.GitOpTypeAPIContent}, enum.GitOpTypeAPIContent, true},
		{"legacy internal flag", Payload{Internal: true}, enum.GitOpTypeAPIContent, true},
		{"legacy external", Payload{}, enum.GitOpTypeGitPush, false},
		{"operation type wins over legacy flag", Payload{Internal: true, OperationType: enum.GitOpTypeGitPush},
			enum.GitOpTypeGitPush, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.payload.RepoID = 3
			tt.payload.PrincipalID = 4
			got := GetInputBaseFromPayload(tt.payload)
			require.Equal(t, types.GithookInputBase{
				RepoID: 3, PrincipalID: 4, Internal: tt.wantInternal, OperationType: tt.wantOp,
			}, got)
		})
	}
}

func TestInfGenerateEnvironmentVariablesForOperation(t *testing.T) {
	ctx := request.WithRequestID(context.Background(), "req-1")

	env, err := GenerateEnvironmentVariablesForOperation(ctx, "http://host/api", 5, 6, false, enum.GitOpTypeGitPush)
	require.NoError(t, err)
	require.NotEmpty(t, env)

	payload, err := hook.LoadPayloadFromMap[Payload](env)
	require.NoError(t, err)
	require.Equal(t, "http://host/api/v1/internal/git-hooks", payload.BaseURL)
	require.EqualValues(t, 5, payload.RepoID)
	require.EqualValues(t, 6, payload.PrincipalID)
	require.Equal(t, "req-1", payload.RequestID)
	require.False(t, payload.Internal)
	require.Equal(t, enum.GitOpTypeGitPush, payload.OperationType)

	// a missing request id is tolerated, API operations are flagged internal.
	env, err = GenerateEnvironmentVariablesForOperation(context.Background(), "http://host/api", 5, 6, true,
		enum.GitOpTypeAPIContent)
	require.NoError(t, err)
	payload, err = hook.LoadPayloadFromMap[Payload](env)
	require.NoError(t, err)
	require.Empty(t, payload.RequestID)
	require.True(t, payload.Internal)
	require.True(t, payload.Disabled)

	_, err = GenerateEnvironmentVariablesForOperation(ctx, "http://host/api", 0, 6, false, enum.GitOpTypeGitPush)
	require.ErrorContains(t, err, "generated payload is invalid")
}

func TestInfCreateWriteParamsForOperation(t *testing.T) {
	ctx := context.Background()
	actor := git.Identity{Name: "Alice", Email: "alice@example.invalid"}

	params, err := CreateWriteParamsForOperation(ctx, "http://host/api", actor, 5, "git-uid", 6, false,
		enum.GitOpTypeGitPush)
	require.NoError(t, err)
	require.Equal(t, actor, params.Actor)
	require.Equal(t, "git-uid", params.RepoUID)
	require.NotEmpty(t, params.EnvVars)

	_, err = CreateWriteParamsForOperation(ctx, "http://host/api", actor, 0, "git-uid", 6, false,
		enum.GitOpTypeGitPush)
	require.ErrorContains(t, err, "failed to generate git hook environment variables")
}

func TestInfRestClientFactory_NewClient(t *testing.T) {
	f := &RestClientFactory{}
	env, err := GenerateEnvironmentVariablesForOperation(context.Background(), "http://host/api", 5, 6, false,
		enum.GitOpTypeGitPush)
	require.NoError(t, err)

	client, err := f.NewClient(env)
	require.NoError(t, err)
	_, isRest := client.(*RestClient)
	require.True(t, isRest)

	disabledEnv, err := GenerateEnvironmentVariablesForOperation(context.Background(), "http://host/api", 5, 6, true,
		enum.GitOpTypeGitPush)
	require.NoError(t, err)
	client, err = f.NewClient(disabledEnv)
	require.NoError(t, err)
	out, err := client.PreReceive(context.Background(), hook.PreReceiveInput{})
	require.NoError(t, err)
	require.Equal(t, []string{"hook disabled"}, out.Messages)

	_, err = f.NewClient(map[string]string{})
	require.Error(t, err)

	// a payload that decodes but doesn't validate is rejected.
	payload := Payload{BaseURL: "http://host"}
	invalidEnv, err := hook.GenerateEnvironmentVariables(payload)
	require.NoError(t, err)
	_, err = f.NewClient(invalidEnv)
	require.ErrorContains(t, err, "payload validation failed")
}

type infHookRequest struct {
	path    string
	headers http.Header
	body    map[string]any
}

func infHookServer(t *testing.T, status int, response string) (*httptest.Server, *infHookRequest) {
	t.Helper()
	got := &infHookRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.headers = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got.body)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func TestInfRestClient_Calls(t *testing.T) {
	ctx := context.Background()
	payload := func(baseURL string) Payload {
		return Payload{
			BaseURL: baseURL + "/hooks/", RepoID: 5, PrincipalID: 6, RequestID: "req-9",
			OperationType: enum.GitOpTypeGitPush, TraceContext: map[string]string{"traceparent": "00-aa-bb-01"},
		}
	}
	errMsg := "denied"
	response, err := json.Marshal(hook.Output{Messages: []string{"hi"}, Error: &errMsg})
	require.NoError(t, err)

	calls := map[string]func(c hook.Client) (hook.Output, error){
		"pre-receive": func(c hook.Client) (hook.Output, error) {
			return c.PreReceive(ctx, hook.PreReceiveInput{})
		},
		"update": func(c hook.Client) (hook.Output, error) {
			return c.Update(ctx, hook.UpdateInput{})
		},
		"post-receive": func(c hook.Client) (hook.Output, error) {
			return c.PostReceive(ctx, hook.PostReceiveInput{})
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			srv, got := infHookServer(t, http.StatusOK, string(response))
			out, err := call(NewRestClient(payload(srv.URL)))
			require.NoError(t, err)
			require.Equal(t, []string{"hi"}, out.Messages)
			require.Equal(t, "denied", *out.Error)

			require.Equal(t, "/hooks/"+name, got.path, "trailing slashes of the base url are trimmed")
			require.Equal(t, "application/json", got.headers.Get("Content-Type"))
			require.Equal(t, "req-9", got.headers.Get(request.HeaderRequestID))
			require.True(t, strings.HasPrefix(got.headers.Get(request.HeaderUserAgent), "Gitness/"))
			require.EqualValues(t, 5, got.body["RepoID"])
			require.EqualValues(t, 6, got.body["PrincipalID"])
		})
	}
}

func TestInfRestClient_ErrorResponses(t *testing.T) {
	ctx := context.Background()
	payload := func(url string) Payload { return Payload{BaseURL: url, RepoID: 1, PrincipalID: 2} }

	t.Run("not found", func(t *testing.T) {
		srv, _ := infHookServer(t, http.StatusNotFound, "")
		_, err := NewRestClient(payload(srv.URL)).PreReceive(ctx, hook.PreReceiveInput{})
		require.ErrorIs(t, err, hook.ErrNotFound)
	})
	t.Run("unexpected status", func(t *testing.T) {
		srv, _ := infHookServer(t, http.StatusInternalServerError, "boom")
		_, err := NewRestClient(payload(srv.URL)).Update(ctx, hook.UpdateInput{})
		require.ErrorContains(t, err, "expected response code 200")
	})
	t.Run("invalid body", func(t *testing.T) {
		srv, _ := infHookServer(t, http.StatusOK, "not json")
		_, err := NewRestClient(payload(srv.URL)).PostReceive(ctx, hook.PostReceiveInput{})
		require.ErrorContains(t, err, "error deserializing response body")
	})
	t.Run("unreachable server", func(t *testing.T) {
		srv, _ := infHookServer(t, http.StatusOK, "{}")
		url := srv.URL
		srv.Close()
		_, err := NewRestClient(payload(url)).PreReceive(ctx, hook.PreReceiveInput{})
		require.ErrorContains(t, err, "request execution failed")
	})
	t.Run("invalid url", func(t *testing.T) {
		_, err := NewRestClient(payload("://bad url")).PreReceive(ctx, hook.PreReceiveInput{})
		require.ErrorContains(t, err, "failed to create new http request")
	})
}

func TestInfUnmarshalResponse_EmptyResponses(t *testing.T) {
	_, err := unmarshalResponse[hook.Output](nil)
	require.ErrorContains(t, err, "http response is empty")

	_, err = unmarshalResponse[hook.Output](&http.Response{StatusCode: http.StatusOK})
	require.ErrorContains(t, err, "http response body is empty")
}

func TestInfLoadFromEnvironment(t *testing.T) {
	// nothing configured
	_, err := LoadFromEnvironment()
	require.Error(t, err)

	env, err := GenerateEnvironmentVariablesForOperation(context.Background(), "http://host/api", 5, 6, false,
		enum.GitOpTypeGitPush)
	require.NoError(t, err)
	for k, v := range env {
		t.Setenv(k, v)
	}
	core, err := LoadFromEnvironment()
	require.NoError(t, err)
	require.NotNil(t, core)
}

func TestInfLoadFromEnvironment_DisabledAndInvalid(t *testing.T) {
	disabled, err := GenerateEnvironmentVariablesForOperation(context.Background(), "http://host/api", 5, 6, true,
		enum.GitOpTypeGitPush)
	require.NoError(t, err)
	for k, v := range disabled {
		t.Setenv(k, v)
	}
	_, err = LoadFromEnvironment()
	require.ErrorIs(t, err, hook.ErrDisabled)

	invalid, err := hook.GenerateEnvironmentVariables(Payload{BaseURL: "http://host"})
	require.NoError(t, err)
	for k, v := range invalid {
		t.Setenv(k, v)
	}
	_, err = LoadFromEnvironment()
	require.ErrorContains(t, err, "payload validation failed")
}
