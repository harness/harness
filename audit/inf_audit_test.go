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

package audit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestInfAction_Validate(t *testing.T) {
	for _, a := range []Action{
		ActionCreated, ActionUpdated, ActionDeleted, ActionUploaded, ActionDownloaded, ActionBypassed, ActionForcePush,
	} {
		require.NoError(t, a.Validate(), a)
	}
	require.ErrorIs(t, Action("").Validate(), ErrActionUndefined)
	require.ErrorIs(t, Action("exploded").Validate(), ErrActionUndefined)
}

func TestInfResourceType_Validate(t *testing.T) {
	for _, rt := range []ResourceType{
		ResourceTypeRepository, ResourceTypeBranchRule, ResourceTypeBranch, ResourceTypeTag, ResourceTypeTagRule,
		ResourceTypePushRule, ResourceTypePullRequest, ResourceTypeRepositorySettings, ResourceTypeSpaceSettings,
		ResourceTypeCodeWebhook, ResourceTypeRegistry, ResourceTypeRegistryUpstreamProxy,
		ResourceTypeRegistryWebhook, ResourceTypeRegistryArtifact,
	} {
		require.NoError(t, rt.Validate(), rt)
	}
	require.ErrorIs(t, ResourceType("").Validate(), ErrResourceTypeUndefined)
	require.ErrorIs(t, ResourceType("unknown").Validate(), ErrResourceTypeUndefined)
}

func TestInfNewResource(t *testing.T) {
	r := NewResource(ResourceTypeRepository, "repo", RepoName, "repo", SpaceName, "space")
	require.Equal(t, ResourceTypeRepository, r.Type)
	require.Equal(t, "repo", r.Identifier)
	require.Equal(t, map[string]string{RepoName: "repo", SpaceName: "space"}, r.Data)

	empty := NewResource(ResourceTypeTag, "v1")
	require.NotNil(t, empty.Data)
	require.Empty(t, empty.Data)
	require.Empty(t, empty.DataAsSlice())

	slice := r.DataAsSlice()
	require.Len(t, slice, 4)
	require.ElementsMatch(t, []string{RepoName, "repo", SpaceName, "space"}, slice)
}

func TestInfResource_Validate(t *testing.T) {
	require.NoError(t, NewResource(ResourceTypeBranch, "main").Validate())
	require.ErrorIs(t, NewResource("bogus", "main").Validate(), ErrResourceTypeUndefined)
	require.ErrorIs(t, NewResource(ResourceTypeBranch, "").Validate(), ErrResourceIdentifierIsRequired)
}

func TestInfEvent_Validate(t *testing.T) {
	valid := func() Event {
		return Event{
			Action:    ActionCreated,
			User:      types.Principal{UID: "alice"},
			SpacePath: "root/space",
			Resource:  NewResource(ResourceTypeRepository, "repo"),
		}
	}
	tests := []struct {
		name    string
		mutate  func(e *Event)
		wantErr error
		wantMsg string
	}{
		{name: "valid", mutate: func(*Event) {}},
		{
			name: "bad action", mutate: func(e *Event) { e.Action = "" },
			wantErr: ErrActionUndefined, wantMsg: "invalid action",
		},
		{name: "no user", mutate: func(e *Event) { e.User.UID = "" }, wantErr: ErrUserIsRequired},
		{name: "no space", mutate: func(e *Event) { e.SpacePath = "" }, wantErr: ErrSpacePathIsRequired},
		{
			name:    "bad resource",
			mutate:  func(e *Event) { e.Resource.Identifier = "" },
			wantErr: ErrResourceIdentifierIsRequired, wantMsg: "invalid resource",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := valid()
			tt.mutate(&e)
			err := e.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantMsg != "" {
				require.ErrorContains(t, err, tt.wantMsg)
			}
		})
	}
}

func TestInfNoopService(t *testing.T) {
	var svc Service = New()
	require.NoError(t, svc.Log(context.Background(), types.Principal{}, Resource{}, ActionCreated, ""))
}

func TestInfEventOptions(t *testing.T) {
	e := &Event{}
	opts := []Option{
		WithID("evt-1"),
		WithNewObject("new"),
		WithOldObject("old"),
		WithClientIP("203.0.113.5"),
		WithRequestMethod(http.MethodPost),
		WithData("k1", "v1"),
		WithData("k2", "v2", "k1", "override"),
	}
	for _, o := range opts {
		o.Apply(e)
	}
	require.Equal(t, "evt-1", e.ID)
	require.Equal(t, DiffObject{OldObject: "old", NewObject: "new"}, e.DiffObject)
	require.Equal(t, "203.0.113.5", e.ClientIP)
	require.Equal(t, http.MethodPost, e.RequestMethod)
	require.Equal(t, map[string]string{"k1": "override", "k2": "v2"}, e.Data)
}

func TestInfRealIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		want       string
	}{
		{"true client ip wins", map[string]string{
			"True-Client-IP": "203.0.113.1", "X-Real-IP": "203.0.113.2", "X-Forwarded-For": "203.0.113.3",
		}, "198.51.100.1:1234", "203.0.113.1"},
		{"x-real-ip next", map[string]string{"X-Real-IP": "203.0.113.2", "X-Forwarded-For": "203.0.113.3"},
			"198.51.100.1:1234", "203.0.113.2"},
		{"first forwarded-for entry", map[string]string{"X-Forwarded-For": "203.0.113.3, 10.0.0.1"},
			"198.51.100.1:1234", "203.0.113.3"},
		{"single forwarded-for entry", map[string]string{"X-Forwarded-For": "203.0.113.3"},
			"198.51.100.1:1234", "203.0.113.3"},
		{"remote address fallback", nil, "198.51.100.1:1234", "198.51.100.1"},
		{"invalid header ip is dropped", map[string]string{"X-Real-IP": "not-an-ip"}, "198.51.100.1:1234", ""},
		{"empty remote address", nil, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			require.Equal(t, tt.want, RealIP(r))
		})
	}
}

func TestInfMiddleware(t *testing.T) {
	var gotCtx context.Context
	handler := Middleware()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotCtx = r.Context()
	}))

	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-Id", "req-42")
	r := httptest.NewRequest(http.MethodPut, "/api/v1/repos", nil)
	r.Header.Set("X-Real-IP", "203.0.113.7")
	handler.ServeHTTP(rec, r)

	require.NotNil(t, gotCtx)
	require.Equal(t, "203.0.113.7", GetRealIP(gotCtx))
	require.Equal(t, "/api/v1/repos", GetPath(gotCtx))
	require.Equal(t, http.MethodPut, GetRequestMethod(gotCtx))
	require.Equal(t, "req-42", GetRequestID(gotCtx))

	// without a resolvable client ip the key is simply not set.
	r = httptest.NewRequest(http.MethodGet, "/x", nil)
	r.RemoteAddr = ""
	handler.ServeHTTP(httptest.NewRecorder(), r)
	require.Empty(t, GetRealIP(gotCtx))
	require.Equal(t, "/x", GetPath(gotCtx))
}

func TestInfSentinelErrorsAreDistinct(t *testing.T) {
	errs := []error{
		ErrActionUndefined, ErrResourceTypeUndefined, ErrResourceIdentifierIsRequired,
		ErrUserIsRequired, ErrSpacePathIsRequired,
	}
	for i, a := range errs {
		for j, b := range errs {
			require.Equal(t, i == j, errors.Is(a, b))
		}
	}
}
