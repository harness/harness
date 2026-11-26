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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGtxAction_Validate(t *testing.T) {
	valid := []Action{
		ActionCreated, ActionUpdated, ActionDeleted, ActionUploaded, ActionDownloaded, ActionBypassed, ActionForcePush,
	}
	for _, a := range valid {
		require.NoError(t, a.Validate(), a)
	}

	for _, a := range []Action{"", "gtx-unknown", "CREATED"} {
		require.ErrorIs(t, a.Validate(), ErrActionUndefined, a)
	}
}

func TestGtxResourceType_Validate(t *testing.T) {
	valid := []ResourceType{
		ResourceTypeRepository, ResourceTypeBranchRule, ResourceTypeBranch, ResourceTypeTag, ResourceTypeTagRule,
		ResourceTypePushRule, ResourceTypePullRequest, ResourceTypeRepositorySettings, ResourceTypeSpaceSettings,
		ResourceTypeCodeWebhook, ResourceTypeRegistry, ResourceTypeRegistryUpstreamProxy, ResourceTypeRegistryWebhook,
		ResourceTypeRegistryArtifact,
	}
	for _, r := range valid {
		require.NoError(t, r.Validate(), r)
	}

	for _, r := range []ResourceType{"", "gtx-unknown", "Repository"} {
		require.ErrorIs(t, r.Validate(), ErrResourceTypeUndefined, r)
	}
}

func TestGtxResource(t *testing.T) {
	t.Run("new resource collects key values", func(t *testing.T) {
		r := NewResource(ResourceTypeRepository, "repo", ResourceName, "name", RepoPath, "/a/b")

		assert.Equal(t, ResourceTypeRepository, r.Type)
		assert.Equal(t, "repo", r.Identifier)
		assert.Equal(t, map[string]string{ResourceName: "name", RepoPath: "/a/b"}, r.Data)
		assert.ElementsMatch(t, []string{ResourceName, "name", RepoPath, "/a/b"}, r.DataAsSlice())
	})

	t.Run("no key values", func(t *testing.T) {
		r := NewResource(ResourceTypeTag, "tag")

		assert.NotNil(t, r.Data)
		assert.Empty(t, r.Data)
		assert.Empty(t, r.DataAsSlice())
	})

	t.Run("validate", func(t *testing.T) {
		require.NoError(t, NewResource(ResourceTypeBranch, "main").Validate())
		require.ErrorIs(t, NewResource("gtx-unknown", "main").Validate(), ErrResourceTypeUndefined)
		require.ErrorIs(t, NewResource(ResourceTypeBranch, "").Validate(), ErrResourceIdentifierIsRequired)
	})
}

func TestGtxEvent_Validate(t *testing.T) {
	valid := func() Event {
		return Event{
			Action:    ActionCreated,
			User:      types.Principal{UID: "admin"},
			SpacePath: "/root/space",
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
			name: "invalid action", mutate: func(e *Event) { e.Action = "gtx" },
			wantErr: ErrActionUndefined, wantMsg: "invalid action",
		},
		{name: "missing user", mutate: func(e *Event) { e.User = types.Principal{} }, wantErr: ErrUserIsRequired},
		{name: "missing space path", mutate: func(e *Event) { e.SpacePath = "" }, wantErr: ErrSpacePathIsRequired},
		{
			name: "invalid resource", mutate: func(e *Event) { e.Resource = NewResource("gtx", "id") },
			wantErr: ErrResourceTypeUndefined, wantMsg: "invalid resource",
		},
		{
			name: "missing resource identifier", mutate: func(e *Event) { e.Resource = NewResource(ResourceTypeTag, "") },
			wantErr: ErrResourceIdentifierIsRequired, wantMsg: "invalid resource",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := valid()
			test.mutate(&e)

			err := e.Validate()
			if test.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, test.wantErr)
			if test.wantMsg != "" {
				require.ErrorContains(t, err, test.wantMsg)
			}
		})
	}
}

func TestGtxOptions(t *testing.T) {
	e := &Event{}

	for _, opt := range []Option{
		WithID("id-1"),
		WithNewObject("new"),
		WithOldObject(42),
		WithClientIP("10.0.0.1"),
		WithRequestMethod(http.MethodPost),
		WithData("k1", "v1"),
		WithData("k2", "v2", "k1", "override"),
	} {
		opt.Apply(e)
	}

	assert.Equal(t, "id-1", e.ID)
	assert.Equal(t, "new", e.DiffObject.NewObject)
	assert.Equal(t, 42, e.DiffObject.OldObject)
	assert.Equal(t, "10.0.0.1", e.ClientIP)
	assert.Equal(t, http.MethodPost, e.RequestMethod)
	assert.Equal(t, map[string]string{"k1": "override", "k2": "v2"}, e.Data)
}

func TestGtxNoopService(t *testing.T) {
	var s Service = New()

	err := s.Log(context.Background(), types.Principal{UID: "admin"},
		NewResource(ResourceTypeRepository, "repo"), ActionCreated, "/root", WithID("x"))
	require.NoError(t, err)

	assert.NotNil(t, ProvideAuditService())
}

func TestGtxRealIP(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		remote  string
		want    string
	}{
		{name: "true client ip has priority", headers: map[string]string{
			"True-Client-IP": "1.1.1.1", "X-Real-IP": "2.2.2.2", "X-Forwarded-For": "3.3.3.3",
		}, remote: "4.4.4.4:80", want: "1.1.1.1"},
		{name: "real ip over forwarded for", headers: map[string]string{
			"X-Real-IP": "2.2.2.2", "X-Forwarded-For": "3.3.3.3",
		}, remote: "4.4.4.4:80", want: "2.2.2.2"},
		{name: "first forwarded for entry", headers: map[string]string{
			"X-Forwarded-For": "3.3.3.3,5.5.5.5",
		}, remote: "4.4.4.4:80", want: "3.3.3.3"},
		{name: "single forwarded for entry", headers: map[string]string{
			"X-Forwarded-For": "3.3.3.3",
		}, remote: "4.4.4.4:80", want: "3.3.3.3"},
		{name: "remote address", remote: "4.4.4.4:8080", want: "4.4.4.4"},
		{name: "remote address without port", remote: "4.4.4.4", want: "4.4.4.4"},
		{
			name: "ipv6 header", headers: map[string]string{"X-Real-IP": "2001:db8::1"},
			remote: "4.4.4.4:80", want: "2001:db8::1",
		},
		{name: "invalid header ip", headers: map[string]string{"X-Real-IP": "not-an-ip"}, remote: "4.4.4.4:80"},
		{name: "empty remote", remote: ""},
		{name: "hostname as remote", remote: "example.com:80"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = test.remote
			for k, v := range test.headers {
				r.Header.Set(k, v)
			}

			assert.Equal(t, test.want, RealIP(r))
		})
	}
}

func TestGtxMiddleware(t *testing.T) {
	var gotCtx context.Context
	handler := Middleware()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotCtx = r.Context()
	}))

	t.Run("fills the context", func(t *testing.T) {
		gotCtx = nil

		w := httptest.NewRecorder()
		w.Header().Set("X-Request-Id", "req-1")
		r := httptest.NewRequest(http.MethodPut, "/api/v1/things?x=1", nil)
		r.Header.Set("X-Real-IP", "9.9.9.9")

		handler.ServeHTTP(w, r)

		require.NotNil(t, gotCtx)
		assert.Equal(t, "9.9.9.9", GetRealIP(gotCtx))
		assert.Equal(t, "/api/v1/things", GetPath(gotCtx))
		assert.Equal(t, http.MethodPut, GetRequestMethod(gotCtx))
		assert.Equal(t, "req-1", GetRequestID(gotCtx))
	})

	t.Run("no ip no request id", func(t *testing.T) {
		gotCtx = nil

		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.RemoteAddr = "garbage"
		handler.ServeHTTP(httptest.NewRecorder(), r)

		require.NotNil(t, gotCtx)
		assert.Empty(t, GetRealIP(gotCtx))
		assert.Equal(t, "/x", GetPath(gotCtx))
		assert.Empty(t, GetRequestID(gotCtx))
	})
}

func TestGtxContextGettersWithWrongTypes(t *testing.T) {
	ctx := context.WithValue(context.Background(), pathKey, 1)
	ctx = context.WithValue(ctx, requestID, 2)
	ctx = context.WithValue(ctx, requestMethod, 3)

	assert.Empty(t, GetPath(ctx))
	assert.Empty(t, GetRequestID(ctx))
	assert.Empty(t, GetRequestMethod(ctx))
}
