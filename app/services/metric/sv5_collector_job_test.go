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

package metric

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	registrystore "github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

type sv5Submitter struct {
	Submitter

	err error
}

func (s *sv5Submitter) SubmitGroups(context.Context) error { return s.err }

type sv5UserStore struct {
	store.PrincipalStore

	users    []*types.User
	listErr  error
	count    int64
	countErr error
}

func (s *sv5UserStore) ListUsers(context.Context, *types.UserFilter) ([]*types.User, error) {
	return s.users, s.listErr
}

func (s *sv5UserStore) CountUsers(context.Context, *types.UserFilter) (int64, error) {
	return s.count, s.countErr
}

type sv5Counter struct{ err error }

func (c sv5Counter) n() (int64, error) { return 5, c.err }

type sv5RepoStore struct {
	store.RepoStore
	sv5Counter
}

func (s sv5RepoStore) Count(context.Context, int64, *types.RepoFilter) (int64, error) { return s.n() }

type sv5PipelineStore struct {
	store.PipelineStore
	sv5Counter
}

func (s sv5PipelineStore) Count(context.Context, int64, *types.ListPipelinesFilter) (int64, error) {
	return s.n()
}

type sv5ExecutionStore struct {
	store.ExecutionStore
	sv5Counter
}

func (s sv5ExecutionStore) Count(context.Context, int64) (int64, error) { return s.n() }

type sv5GitspaceStore struct {
	store.GitspaceConfigStore
	sv5Counter
}

func (s sv5GitspaceStore) Count(context.Context, *types.GitspaceFilter) (int64, error) { return s.n() }

type sv5RegistryStore struct {
	registrystore.RegistryRepository
	sv5Counter
}

func (s sv5RegistryStore) Count(context.Context) (int64, error) { return s.n() }

type sv5ArtifactStore struct {
	registrystore.ArtifactRepository
	sv5Counter
}

func (s sv5ArtifactStore) Count(context.Context) (int64, error) { return s.n() }

type sv5Errs struct {
	submit, users, count, repo, pipeline, exec, gitspace, registry, artifact error
	noUsers                                                                  bool
}

func sv5NewJob(e sv5Errs, endpoint string, enabled bool) *CollectorJob {
	users := []*types.User{{Email: "admin@example.com", Created: 1700000000000}}
	if e.noUsers {
		users = nil
	}
	return NewCollectorJob(
		&Values{Enabled: enabled, Hostname: "host", InstallID: "install"},
		endpoint, "tok",
		&sv5UserStore{users: users, listErr: e.users, count: 3, countErr: e.count},
		sv5RepoStore{sv5Counter: sv5Counter{e.repo}},
		sv5PipelineStore{sv5Counter: sv5Counter{e.pipeline}},
		sv5ExecutionStore{sv5Counter: sv5Counter{e.exec}},
		nil,
		sv5GitspaceStore{sv5Counter: sv5Counter{e.gitspace}},
		sv5RegistryStore{sv5Counter: sv5Counter{e.registry}},
		sv5ArtifactStore{sv5Counter: sv5Counter{e.artifact}},
		&sv5Submitter{err: e.submit},
	)
}

func TestSv5CollectorRegisterDisabled(t *testing.T) {
	// A disabled collector must not touch the (nil) scheduler.
	require.NoError(t, sv5NewJob(sv5Errs{}, "", false).Register(context.Background()))

	res, err := sv5NewJob(sv5Errs{}, "", false).Handle(context.Background(), "", nil)
	require.NoError(t, err)
	require.Empty(t, res)
}

func TestSv5CollectorHandleErrors(t *testing.T) {
	tests := []struct {
		name string
		errs sv5Errs
		msg  string
	}{
		{"submit", sv5Errs{submit: errSv5Boom}, "submit metric groups"},
		{"list users", sv5Errs{users: errSv5Boom}, "boom"},
		{"count users", sv5Errs{count: errSv5Boom}, "users count"},
		{"repos", sv5Errs{repo: errSv5Boom}, "repositories count"},
		{"pipelines", sv5Errs{pipeline: errSv5Boom}, "pipelines count"},
		{"executions", sv5Errs{exec: errSv5Boom}, "executions count"},
		{"gitspaces", sv5Errs{gitspace: errSv5Boom}, "gitspace count"},
		{"registries", sv5Errs{registry: errSv5Boom}, "registries count"},
		{"artifacts", sv5Errs{artifact: errSv5Boom}, "artifacts count"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sv5NewJob(tc.errs, "http://127.0.0.1:1", true).Handle(context.Background(), "", nil)
			require.ErrorIs(t, err, errSv5Boom)
			require.ErrorContains(t, err, tc.msg)
		})
	}

	res, err := sv5NewJob(sv5Errs{noUsers: true}, "http://127.0.0.1:1", true).Handle(context.Background(), "", nil)
	require.NoError(t, err)
	require.Empty(t, res)
}

func TestSv5CollectorHandleSends(t *testing.T) {
	var gotKey, gotType string
	var got metricData
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.URL.Query().Get("api_key")
		gotType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	res, err := sv5NewJob(sv5Errs{}, srv.URL, true).Handle(context.Background(), "", nil)
	require.NoError(t, err)
	require.Equal(t, "202 Accepted", res)
	require.Equal(t, "tok", gotKey)
	require.Equal(t, "application/json", gotType)
	require.Equal(t, "admin@example.com", got.Installer)
	require.Equal(t, int64(3), got.Users)
	require.Equal(t, int64(5), got.RepoCount)
	require.Equal(t, int64(5), got.ArtifactCount)
	require.Equal(t, "install", got.InstallID)

	_, err = sv5NewJob(sv5Errs{}, "http://127.0.0.1:1", true).Handle(context.Background(), "", nil)
	require.ErrorContains(t, err, "failed to send metric data")

	_, err = sv5NewJob(sv5Errs{}, "http://bad host", true).Handle(context.Background(), "", nil)
	require.ErrorContains(t, err, "failed to create a request")
}

var _ job.Handler = (*CollectorJob)(nil)
