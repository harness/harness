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

package exporter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

type sv5JobStore struct {
	job.Store

	progress  []*job.Job
	listErr   error
	deleteErr error
	deleted   int64
	created   []*job.Job
	createErr error
}

func (s *sv5JobStore) ListByGroupID(context.Context, string) ([]*job.Job, error) {
	return s.progress, s.listErr
}

func (s *sv5JobStore) DeleteByGroupID(context.Context, string) (int64, error) {
	return s.deleted, s.deleteErr
}

func (s *sv5JobStore) Create(_ context.Context, j *job.Job) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, j)
	return nil
}

type sv5PublicAccess struct {
	publicaccess.Service

	public bool
	err    error
}

func (p *sv5PublicAccess) Get(context.Context, enum.PublicResourceType, string) (bool, error) {
	return p.public, p.err
}

type sv5RepoStore struct {
	store.RepoStore

	repo *types.Repository
	err  error
}

func (r *sv5RepoStore) Find(context.Context, int64) (*types.Repository, error) {
	return r.repo, r.err
}

type sv5Encrypter struct {
	encrypt.Encrypter

	encErr error
	decErr error
}

func (e *sv5Encrypter) Encrypt(p string) ([]byte, error) {
	if e.encErr != nil {
		return nil, e.encErr
	}
	return []byte(p), nil
}

func (e *sv5Encrypter) Decrypt(c []byte) (string, error) {
	if e.decErr != nil {
		return "", e.decErr
	}
	return string(c), nil
}

func sv5NewScheduler(t *testing.T, st job.Store) *job.Scheduler {
	t.Helper()
	s, err := job.NewScheduler(st, nil, nil, nil, "inst", 1, time.Hour)
	require.NoError(t, err)
	return s
}

func sv5Encode(t *testing.T, in Input) string {
	t.Helper()
	b, err := json.Marshal(in)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(b)
}

func TestSv5ModifyURL(t *testing.T) {
	got, err := modifyURL("https://example.com/org/repo.git", "tok")
	require.NoError(t, err)
	require.Equal(t, "https://token:tok@example.com/org/repo.git", got)

	_, err = modifyURL("://bad", "tok")
	require.Error(t, err)
}

func TestSv5CheckJobAlreadyRunning(t *testing.T) {
	require.NoError(t, checkJobAlreadyRunning(nil))
	require.NoError(t, checkJobAlreadyRunning([]job.Progress{{State: job.JobStateFinished}}))
	require.ErrorIs(t, checkJobAlreadyRunning(
		[]job.Progress{{State: job.JobStateFinished}, {State: job.JobStateRunning}}), ErrJobRunning)
}

func TestSv5JobGroupID(t *testing.T) {
	require.Equal(t, "export_space_7", getJobGroupID(7))
}

func TestSv5GetJobInput(t *testing.T) {
	r := &Repository{encrypter: &sv5Encrypter{}}
	in := Input{Identifier: "r1", ID: 3, IsPublic: true, HarnessCodeInfo: HarnessCodeInfo{AccountID: "a"}}

	got, err := r.getJobInput(sv5Encode(t, in))
	require.NoError(t, err)
	require.Equal(t, in, got)

	_, err = r.getJobInput("!!!not-base64")
	require.ErrorContains(t, err, "base64")

	r.encrypter = &sv5Encrypter{decErr: errSv5Boom}
	_, err = r.getJobInput(sv5Encode(t, in))
	require.ErrorIs(t, err, errSv5Boom)

	r.encrypter = &sv5Encrypter{}
	_, err = r.getJobInput(base64.StdEncoding.EncodeToString([]byte("{not json")))
	require.ErrorContains(t, err, "unmarshal")
}

func TestSv5HandleErrors(t *testing.T) {
	ctx := context.Background()
	r := &Repository{encrypter: &sv5Encrypter{}}

	_, err := r.Handle(ctx, "!!!", nil)
	require.Error(t, err)

	// Missing token makes the client constructor fail.
	_, err = r.Handle(ctx, sv5Encode(t, Input{HarnessCodeInfo: HarnessCodeInfo{AccountID: "a"}}), nil)
	require.ErrorContains(t, err, "orgId required")

	r.repoStore = &sv5RepoStore{err: errSv5Boom}
	full := Input{HarnessCodeInfo: HarnessCodeInfo{
		AccountID: "a", OrgIdentifier: "o", ProjectIdentifier: "p", Token: "t",
	}}
	_, err = r.Handle(ctx, sv5Encode(t, full), nil)
	require.ErrorIs(t, err, errSv5Boom)
}

func TestSv5RunManyForSpace(t *testing.T) {
	ctx := context.Background()
	info := &HarnessCodeInfo{AccountID: "a"}
	repos := []*types.Repository{{ID: 1, Identifier: "one", Path: "s/one"}, {ID: 2, Identifier: "two", Path: "s/two"}}

	t.Run("list error", func(t *testing.T) {
		r := &Repository{scheduler: sv5NewScheduler(t, &sv5JobStore{listErr: errSv5Boom})}
		require.ErrorIs(t, r.RunManyForSpace(ctx, 1, repos, info), errSv5Boom)
	})

	t.Run("job running", func(t *testing.T) {
		st := &sv5JobStore{progress: []*job.Job{{State: job.JobStateRunning}}}
		r := &Repository{scheduler: sv5NewScheduler(t, st)}
		require.ErrorIs(t, r.RunManyForSpace(ctx, 1, repos, info), ErrJobRunning)
	})

	t.Run("purge error", func(t *testing.T) {
		st := &sv5JobStore{progress: []*job.Job{{State: job.JobStateFinished}}, deleteErr: errSv5Boom}
		r := &Repository{scheduler: sv5NewScheduler(t, st)}
		require.ErrorIs(t, r.RunManyForSpace(ctx, 1, repos, info), errSv5Boom)
	})

	t.Run("public access error", func(t *testing.T) {
		r := &Repository{
			scheduler:    sv5NewScheduler(t, &sv5JobStore{}),
			publicAccess: &sv5PublicAccess{err: errSv5Boom},
		}
		require.ErrorIs(t, r.RunManyForSpace(ctx, 1, repos, info), errSv5Boom)
	})

	t.Run("encrypt error", func(t *testing.T) {
		r := &Repository{
			scheduler:    sv5NewScheduler(t, &sv5JobStore{}),
			publicAccess: &sv5PublicAccess{},
			encrypter:    &sv5Encrypter{encErr: errSv5Boom},
		}
		require.ErrorIs(t, r.RunManyForSpace(ctx, 1, repos, info), errSv5Boom)
	})

	t.Run("success after purge", func(t *testing.T) {
		st := &sv5JobStore{progress: []*job.Job{{State: job.JobStateFinished}}, deleted: 1}
		r := &Repository{
			scheduler:    sv5NewScheduler(t, st),
			publicAccess: &sv5PublicAccess{public: true},
			encrypter:    &sv5Encrypter{},
		}
		require.NoError(t, r.RunManyForSpace(ctx, 1, repos, info))
		require.Len(t, st.created, 2)
		require.Equal(t, "export_repo_1", st.created[0].UID)
		require.Equal(t, jobType, st.created[0].Type)

		r.encrypter = &sv5Encrypter{}
		got, err := r.getJobInput(st.created[1].Data)
		require.NoError(t, err)
		require.Equal(t, int64(2), got.ID)
		require.True(t, got.IsPublic)
	})
}

func TestSv5GetProgressForSpace(t *testing.T) {
	ctx := context.Background()

	r := &Repository{scheduler: sv5NewScheduler(t, &sv5JobStore{listErr: errSv5Boom})}
	_, err := r.GetProgressForSpace(ctx, 1)
	require.ErrorIs(t, err, errSv5Boom)

	r = &Repository{scheduler: sv5NewScheduler(t, &sv5JobStore{})}
	_, err = r.GetProgressForSpace(ctx, 1)
	require.ErrorIs(t, err, ErrNotFound)

	r = &Repository{scheduler: sv5NewScheduler(t, &sv5JobStore{progress: []*job.Job{{State: job.JobStateFinished}}})}
	p, err := r.GetProgressForSpace(ctx, 1)
	require.NoError(t, err)
	require.Len(t, p, 1)
}

func TestSv5NewClientValidation(t *testing.T) {
	tests := []struct {
		name                                  string
		base, account, org, project, token, e string
	}{
		{"base", "", "a", "o", "p", "t", "baseUrl"},
		{"account", "http://x", "", "o", "p", "t", "accountID"},
		{"org", "http://x", "a", "", "p", "t", "orgId"},
		{"project", "http://x", "a", "o", "", "t", "projectId"},
		{"token", "http://x", "a", "o", "p", "", "token"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newHarnessCodeClient(tc.base, tc.account, tc.org, tc.project, tc.token, false)
			require.ErrorContains(t, err, tc.e)
		})
	}

	c, err := newHarnessCodeClient("http://x", "a", "o", "p", "t", true)
	require.NoError(t, err)
	require.NotNil(t, c.client)
}

func TestSv5HarnessCodeClientCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	status := http.StatusOK
	var gotMethod, gotPath, gotKey, gotRouting string
	var gotBody repo.CreateInput

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotKey = r.Header.Get(headerAPIKey)
		gotRouting = r.URL.Query().Get(routingID)
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}
		w.WriteHeader(status)
		if status < 300 && r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"identifier":"newrepo","git_url":"https://h/x.git"}`))
		}
	}))
	defer srv.Close()

	c, err := newHarnessCodeClient(srv.URL+"/", "acc", "org", "proj", "secret", false)
	require.NoError(t, err)

	created, err := c.CreateRepo(ctx, repo.CreateInput{Identifier: "newrepo", Description: "d"})
	require.NoError(t, err)
	require.Equal(t, "newrepo", created.Identifier)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/v1/accounts/acc/orgs/org/projects/proj/repos", gotPath)
	require.Equal(t, "secret", gotKey)
	require.Equal(t, "acc", gotRouting)
	require.Equal(t, "newrepo", gotBody.Identifier)

	require.NoError(t, c.DeleteRepo(ctx, "newrepo"))
	require.Equal(t, http.MethodDelete, gotMethod)
	require.Equal(t, "/v1/accounts/acc/orgs/org/projects/proj/repos/newrepo", gotPath)

	status = http.StatusConflict
	_, err = c.CreateRepo(ctx, repo.CreateInput{})
	require.ErrorIs(t, err, errHTTPDuplicate)
	status = http.StatusNotFound
	require.ErrorIs(t, c.DeleteRepo(ctx, "x"), errHTTPNotFound)

	// Invalid JSON in a successful response.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer bad.Close()
	c2, err := newHarnessCodeClient(bad.URL, "a", "o", "p", "t", false)
	require.NoError(t, err)
	_, err = c2.CreateRepo(ctx, repo.CreateInput{})
	require.ErrorContains(t, err, "deserializing")

	// Unreachable server.
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()
	c3, err := newHarnessCodeClient(closed.URL, "a", "o", "p", "t", false)
	require.NoError(t, err)
	_, err = c3.CreateRepo(ctx, repo.CreateInput{})
	require.ErrorContains(t, err, "request execution failed")
	require.ErrorContains(t, c3.DeleteRepo(ctx, "x"), "request execution failed")
}

func TestSv5MapStatusCodeToError(t *testing.T) {
	tests := []struct {
		code int
		err  error
		msg  string
	}{
		{200, nil, ""},
		{500, errHTTPInternal, ""},
		{503, nil, "server side"},
		{404, errHTTPNotFound, ""},
		{400, errHTTPBadRequest, ""},
		{409, errHTTPDuplicate, ""},
		{403, nil, "client side"},
		{302, nil, "further action"},
	}
	for _, tc := range tests {
		err := mapStatusCodeToError(tc.code)
		switch {
		case tc.err != nil:
			require.ErrorIs(t, err, tc.err)
		case tc.msg != "":
			require.ErrorContains(t, err, tc.msg)
		default:
			require.NoError(t, err)
		}
	}
}

func TestSv5HelpersAppendPathAndUnmarshal(t *testing.T) {
	require.Equal(t, "http://x", appendPath("http://x", ""))
	require.Equal(t, "http://x/a/b", appendPath("http://x//", "//a/b"))

	require.ErrorContains(t, unmarshalResponse(nil, nil), "empty")
}
